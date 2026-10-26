package service

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"

	"tourna-desk/internal/models"
)

// TeamImportSummary represents the result of a bulk team Excel (.xlsx / .xls) import.
type TeamImportSummary struct {
	ImportedCount int           `json:"imported_count"`
	SkippedCount  int           `json:"skipped_count"`
	Teams         []models.Team `json:"teams"`
}

// gsbHeaderColumns stores the dynamically detected column indices from an "Okul Taahhüt Listesi" or template sheet.
type gsbHeaderColumns struct {
	Found           bool
	HeaderRowIdx    int
	SchoolCol       int // Primary - Required ("Okul Adı", "Okul", "School Name")
	CityCol         int // Optional ("İl")
	DistrictCol     int // Optional/Auxiliary ("İlçe")
	SeasonCol       int // Optional ("Sezon")
	SportCol        int // Optional ("Spor Dalı")
	CategoryTypeCol int // Optional ("Kategori Türü")
	GenderCol       int // Optional ("Cinsiyet")
}

// ImportTeamsFromExcel preserves the existing service signature while supporting both .xlsx and legacy .xls files.
func (s *TournamentService) ImportTeamsFromExcel(categoryID int64, r io.Reader) ([]models.Team, error) {
	summary, err := s.ImportTeamsWithSummary(categoryID, "", r)
	if err != nil {
		return nil, err
	}
	return summary.Teams, nil
}

// ImportTeamsWithSummary parses an uploaded .xlsx or legacy .xls file (including official GSB "Okul Taahhüt Listesi"
// exports with top metadata rows), prevents duplicates within the category, and returns a detailed import summary.
func (s *TournamentService) ImportTeamsWithSummary(categoryID int64, filename string, r io.Reader) (*TeamImportSummary, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("uploaded excel file is empty")
	}

	rows, err := readSpreadsheetRows(filename, data)
	if err != nil {
		return nil, err
	}

	// Verify there is at least one non-empty row in the sheet
	firstNonEmptyIdx := -1
	for i, row := range rows {
		if !isRowEmpty(row) {
			firstNonEmptyIdx = i
			break
		}
	}
	if firstNonEmptyIdx == -1 {
		return nil, errors.New("excel sheet is empty")
	}

	// 1. Dynamic Header Scanner
	cols := detectGSBHeaderColumns(rows, firstNonEmptyIdx)
	startRowIdx := firstNonEmptyIdx
	if cols.Found {
		startRowIdx = cols.HeaderRowIdx + 1
	}

	// 2. Extract raw rows beneath header
	type extractedCandidate struct {
		schoolName string
		district   string
	}
	var candidates []extractedCandidate
	skippedCount := 0

	for i := startRowIdx; i < len(rows); i++ {
		row := rows[i]
		if isRowEmpty(row) {
			skippedCount++
			continue
		}

		var rawSchool string
		if cols.SchoolCol >= 0 && cols.SchoolCol < len(row) {
			rawSchool = cleanWhitespace(row[cols.SchoolCol])
		}
		if rawSchool == "" {
			skippedCount++
			continue
		}

		// Skip stray repeated header rows or ministry footer/title lines if any
		if !cols.Found && isMinistryBannerRow(rawSchool) {
			skippedCount++
			continue
		}
		if isPrimarySchoolHeader(normalizeHeaderKey(rawSchool)) {
			skippedCount++
			continue
		}

		var district string
		if cols.DistrictCol >= 0 && cols.DistrictCol < len(row) {
			district = cleanWhitespace(row[cols.DistrictCol])
		}

		candidates = append(candidates, extractedCandidate{
			schoolName: rawSchool,
			district:   district,
		})
	}

	// Load existing teams in this category for duplicate prevention & district disambiguation
	existingTeams, err := s.listTeamsByCategory(categoryID)
	if err != nil {
		return nil, err
	}

	existingNameSet := make(map[string]bool, len(existingTeams)*2)
	for _, tm := range existingTeams {
		if k := normalizeSchoolIdentity(tm.SchoolName); k != "" {
			existingNameSet[k] = true
		}
		if k := normalizeSchoolIdentity(tm.DisplayName); k != "" {
			existingNameSet[k] = true
		}
	}

	// Determine if two schools from DIFFERENT districts share the exact same Okul Adı in the import batch
	districtsBySchool := make(map[string]map[string]bool)
	for _, c := range candidates {
		sKey := normalizeSchoolIdentity(c.schoolName)
		dKey := normalizeSchoolIdentity(c.district)
		if sKey == "" || dKey == "" {
			continue
		}
		if districtsBySchool[sKey] == nil {
			districtsBySchool[sKey] = make(map[string]bool)
		}
		districtsBySchool[sKey][dKey] = true
	}

	createdTeams := make([]models.Team, 0, len(candidates))
	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for _, c := range candidates {
		finalName := c.schoolName
		sKey := normalizeSchoolIdentity(c.schoolName)
		dKey := normalizeSchoolIdentity(c.district)

		// Optional disambiguation: if multiple schools from different districts share the exact same name,
		// append the district name e.g. "Atatürk Ortaokulu (Gülnar)" unless already included.
		if c.district != "" && len(districtsBySchool[sKey]) > 1 {
			if !strings.Contains(normalizeSchoolIdentity(finalName), dKey) {
				finalName = fmt.Sprintf("%s (%s)", c.schoolName, c.district)
			}
		}

		finalKey := normalizeSchoolIdentity(finalName)
		if finalKey == "" || existingNameSet[finalKey] {
			skippedCount++
			continue
		}

		res, err := tx.Exec(`
			INSERT INTO teams (category_id, stage_id, school_name)
			VALUES (?, NULL, ?)`,
			categoryID, finalName,
		)
		if err != nil {
			return nil, err
		}
		id, _ := res.LastInsertId()

		existingNameSet[finalKey] = true
		createdTeams = append(createdTeams, models.Team{
			ID:          id,
			CategoryID:  categoryID,
			StageID:     nil,
			SchoolName:  finalName,
			DisplayName: models.FormatTeamDisplayName(finalName),
			CreatedAt:   time.Now(),
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &TeamImportSummary{
		ImportedCount: len(createdTeams),
		SkippedCount:  skippedCount,
		Teams:         createdTeams,
	}, nil
}

// ============================================================================
// DUAL ENGINE SPREADSHEET READER (.xlsx via excelize + legacy .xls via pure Go)
// ============================================================================

var (
	ole2Magic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	zipMagic  = []byte{0x50, 0x4B, 0x03, 0x04}
)

func readSpreadsheetRows(filename string, data []byte) ([][]string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))

	isOLE2 := len(data) >= 8 && bytes.Equal(data[:8], ole2Magic)
	isZIP := len(data) >= 4 && bytes.Equal(data[:4], zipMagic)

	// 1. If explicitly .xls or starts with OLE2 Compound File Binary header
	if isOLE2 || (ext == ".xls" && !isZIP) {
		if isOLE2 {
			return readXLSBIFFRows(data)
		}
		// Some Ministry / e-Okul exports save HTML tables or XML SpreadsheetML with a .xls extension
		if looksLikeHTMLOrXML(data) {
			if rows, err := readHTMLOrXMLTableRows(data); err == nil && len(rows) > 0 {
				return rows, nil
			}
		}
		// Try BIFF reader anyway, then fallback to excelize
		if rows, err := readXLSBIFFRows(data); err == nil {
			return rows, nil
		}
	}

	// 2. If .xlsx or ZIP container
	if isZIP || ext == ".xlsx" || ext == "" {
		rows, err := readXLSXRows(data)
		if err == nil {
			return rows, nil
		}
		// Fallback to .xls readers if a legacy .xls was uploaded without filename or with wrong extension
		if isOLE2 {
			return readXLSBIFFRows(data)
		}
		if looksLikeHTMLOrXML(data) {
			if htmlRows, htmlErr := readHTMLOrXMLTableRows(data); htmlErr == nil && len(htmlRows) > 0 {
				return htmlRows, nil
			}
		}
		return nil, fmt.Errorf("invalid excel file: %w", err)
	}

	// 3. Final fallback chain
	if rows, err := readXLSXRows(data); err == nil {
		return rows, nil
	}
	if rows, err := readXLSBIFFRows(data); err == nil {
		return rows, nil
	}
	if looksLikeHTMLOrXML(data) {
		if rows, err := readHTMLOrXMLTableRows(data); err == nil && len(rows) > 0 {
			return rows, nil
		}
	}
	return nil, errors.New("unsupported or corrupted Excel file (.xlsx / .xls)")
}

func readXLSXRows(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, errors.New("excel file contains no sheets")
	}
	return f.GetRows(sheetName)
}

// ============================================================================
// PURE GO BIFF8 / BIFF5 LEGACY .XLS PARSER (via github.com/richardlehane/mscfb)
// ============================================================================

const (
	biffRecBOF2     uint16 = 0x0009
	biffRecBOF3     uint16 = 0x0209
	biffRecBOF4     uint16 = 0x0409
	biffRecBOF8     uint16 = 0x0809
	biffRecEOF      uint16 = 0x000A
	biffRecContinue uint16 = 0x003C
	biffRecCodepage uint16 = 0x0042
	biffRecSST      uint16 = 0x00FC
	biffRecLabelSST uint16 = 0x00FD
	biffRecLabel    uint16 = 0x0204
	biffRecRString  uint16 = 0x00D6
	biffRecRK       uint16 = 0x027E
	biffRecMulRK    uint16 = 0x00BD
	biffRecNumber   uint16 = 0x0203
	biffRecFormula  uint16 = 0x0006
	biffRecString   uint16 = 0x0207
)

type biffRecord struct {
	id   uint16
	data []byte
}

func readXLSBIFFRows(data []byte) ([][]string, error) {
	doc, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid .xls compound file: %w", err)
	}

	var workbookStream []byte
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		nameLower := strings.ToLower(strings.TrimSpace(entry.Name))
		if nameLower == "workbook" || nameLower == "book" {
			workbookStream, err = io.ReadAll(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to read .xls workbook stream: %w", err)
			}
			break
		}
	}

	if len(workbookStream) == 0 {
		return nil, errors.New("no Workbook/Book stream found in .xls file")
	}

	records := parseBIFFRecords(workbookStream)
	if len(records) == 0 {
		return nil, errors.New("empty BIFF stream in .xls file")
	}

	codepage := uint16(1254) // Default to Turkish Windows-1254 for 8-bit legacy strings
	biffVersion := uint16(0x0600)
	var sst []string

	// Pass 1: Extract global Codepage and Shared String Table (SST + CONTINUE fragments)
	for i := 0; i < len(records); i++ {
		rec := records[i]
		switch rec.id {
		case biffRecBOF8, biffRecBOF4, biffRecBOF3, biffRecBOF2:
			if len(rec.data) >= 4 {
				ver := binary.LittleEndian.Uint16(rec.data[0:2])
				subType := binary.LittleEndian.Uint16(rec.data[2:4])
				if subType == 0x0005 && ver != 0 {
					biffVersion = ver
				}
			}
		case biffRecCodepage:
			if len(rec.data) >= 2 {
				codepage = binary.LittleEndian.Uint16(rec.data[0:2])
			}
		case biffRecSST:
			fragments := [][]byte{rec.data}
			for i+1 < len(records) && records[i+1].id == biffRecContinue {
				i++
				fragments = append(fragments, records[i].data)
			}
			sst = parseBIFF8SST(fragments, codepage)
		}
	}

	// Pass 2: Extract Worksheet substreams (substreamType == 0x0010)
	inSheet := false
	sheetVersion := biffVersion
	cells := make(map[int]map[int]string)
	maxRow := -1
	maxCol := -1
	lastFormulaRow := -1
	lastFormulaCol := -1

	setCell := func(r, c int, val string) {
		if r < 0 || c < 0 {
			return
		}
		if cells[r] == nil {
			cells[r] = make(map[int]string)
		}
		cells[r][c] = val
		if r > maxRow {
			maxRow = r
		}
		if c > maxCol {
			maxCol = c
		}
	}

	for i := 0; i < len(records); i++ {
		rec := records[i]
		switch rec.id {
		case biffRecBOF8, biffRecBOF4, biffRecBOF3, biffRecBOF2:
			if len(rec.data) >= 4 {
				ver := binary.LittleEndian.Uint16(rec.data[0:2])
				subType := binary.LittleEndian.Uint16(rec.data[2:4])
				if subType == 0x0010 {
					inSheet = true
					if ver != 0 {
						sheetVersion = ver
					}
				} else {
					inSheet = false
				}
			}
		case biffRecEOF:
			if inSheet {
				inSheet = false
				// If we already populated rows from the first non-empty worksheet, stop here
				if maxRow >= 0 {
					goto doneSheets
				}
			}
		default:
			if !inSheet {
				continue
			}
			switch rec.id {
			case biffRecLabelSST:
				if len(rec.data) >= 10 {
					r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
					c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
					sstIdx := int(binary.LittleEndian.Uint32(rec.data[6:10]))
					if sstIdx >= 0 && sstIdx < len(sst) {
						setCell(r, c, sst[sstIdx])
					}
				}
			case biffRecLabel, biffRecRString:
				if len(rec.data) > 6 {
					r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
					c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
					val := decodeBIFFCellString(rec.data[6:], sheetVersion, codepage)
					setCell(r, c, val)
				}
			case biffRecRK:
				if len(rec.data) >= 10 {
					r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
					c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
					rk := binary.LittleEndian.Uint32(rec.data[6:10])
					setCell(r, c, formatRKNumber(rk))
				}
			case biffRecMulRK:
				if len(rec.data) >= 6 {
					r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
					firstCol := int(binary.LittleEndian.Uint16(rec.data[2:4]))
					pos := 4
					col := firstCol
					for pos+6 <= len(rec.data)-2 {
						rk := binary.LittleEndian.Uint32(rec.data[pos+2 : pos+6])
						setCell(r, col, formatRKNumber(rk))
						pos += 6
						col++
					}
				}
			case biffRecNumber:
				if len(rec.data) >= 14 {
					r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
					c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
					bits := binary.LittleEndian.Uint64(rec.data[6:14])
					val := math.Float64frombits(bits)
					setCell(r, c, strconv.FormatFloat(val, 'f', -1, 64))
				}
			case biffRecFormula:
				if len(rec.data) >= 6 {
					lastFormulaRow = int(binary.LittleEndian.Uint16(rec.data[0:2]))
					lastFormulaCol = int(binary.LittleEndian.Uint16(rec.data[2:4]))
				}
			case biffRecString:
				if lastFormulaRow >= 0 && lastFormulaCol >= 0 && len(rec.data) > 0 {
					val := decodeBIFFCellString(rec.data, sheetVersion, codepage)
					setCell(lastFormulaRow, lastFormulaCol, val)
					lastFormulaRow = -1
					lastFormulaCol = -1
				}
			}
		}
	}

doneSheets:
	if maxRow < 0 || maxCol < 0 {
		return [][]string{}, nil
	}

	grid := make([][]string, maxRow+1)
	for r := 0; r <= maxRow; r++ {
		rowMap, ok := cells[r]
		if !ok {
			grid[r] = []string{}
			continue
		}
		rowSlice := make([]string, maxCol+1)
		for c, val := range rowMap {
			if c >= 0 && c <= maxCol {
				rowSlice[c] = val
			}
		}
		grid[r] = rowSlice
	}
	return grid, nil
}

func parseBIFFRecords(stream []byte) []biffRecord {
	var records []biffRecord
	offset := 0
	for offset+4 <= len(stream) {
		recID := binary.LittleEndian.Uint16(stream[offset : offset+2])
		recLen := int(binary.LittleEndian.Uint16(stream[offset+2 : offset+4]))
		offset += 4
		if offset+recLen > len(stream) {
			recLen = len(stream) - offset
		}
		records = append(records, biffRecord{
			id:   recID,
			data: stream[offset : offset+recLen],
		})
		offset += recLen
	}
	return records
}

// parseBIFF8SST parses the BIFF8 Shared String Table across SST and CONTINUE record boundaries.
func parseBIFF8SST(fragments [][]byte, codepage uint16) []string {
	if len(fragments) == 0 || len(fragments[0]) < 8 {
		return nil
	}
	uniqueStrings := int(binary.LittleEndian.Uint32(fragments[0][4:8]))
	if uniqueStrings <= 0 {
		return nil
	}
	if uniqueStrings > 500000 {
		uniqueStrings = 500000
	}

	fragIdx := 0
	pos := 8

	advanceIfEnd := func() bool {
		for fragIdx < len(fragments) && pos >= len(fragments[fragIdx]) {
			fragIdx++
			pos = 0
		}
		return fragIdx < len(fragments)
	}

	readByte := func() (byte, bool) {
		if !advanceIfEnd() {
			return 0, false
		}
		b := fragments[fragIdx][pos]
		pos++
		return b, true
	}

	readUint16 := func() (uint16, bool) {
		b0, ok := readByte()
		if !ok {
			return 0, false
		}
		b1, ok := readByte()
		if !ok {
			return 0, false
		}
		return uint16(b0) | (uint16(b1) << 8), true
	}

	readUint32 := func() (uint32, bool) {
		w0, ok := readUint16()
		if !ok {
			return 0, false
		}
		w1, ok := readUint16()
		if !ok {
			return 0, false
		}
		return uint32(w0) | (uint32(w1) << 16), true
	}

	skipBytes := func(n int) bool {
		rem := n
		for rem > 0 {
			if !advanceIfEnd() {
				return false
			}
			avail := len(fragments[fragIdx]) - pos
			if rem <= avail {
				pos += rem
				return true
			}
			rem -= avail
			fragIdx++
			pos = 0
		}
		return true
	}

	result := make([]string, 0, uniqueStrings)
	for sIdx := 0; sIdx < uniqueStrings; sIdx++ {
		if !advanceIfEnd() {
			break
		}
		charCount, ok := readUint16()
		if !ok {
			break
		}
		flags, ok := readByte()
		if !ok {
			break
		}

		is16Bit := (flags & 0x01) != 0
		hasExt := (flags & 0x04) != 0
		hasRich := (flags & 0x08) != 0

		var richRuns uint16
		if hasRich {
			richRuns, ok = readUint16()
			if !ok {
				break
			}
		}
		var extSize uint32
		if hasExt {
			extSize, ok = readUint32()
			if !ok {
				break
			}
		}

		var sb strings.Builder
		remChars := int(charCount)
		firstChunk := true

		for remChars > 0 {
			if !firstChunk || pos >= len(fragments[fragIdx]) {
				if !advanceIfEnd() {
					remChars = 0
					break
				}
				// When a string spans into a new CONTINUE fragment, byte 0 of the fragment is the new option flags byte
				if !firstChunk && pos == 0 {
					contFlags, ok := readByte()
					if !ok {
						break
					}
					is16Bit = (contFlags & 0x01) != 0
				}
			}
			firstChunk = false

			availBytes := len(fragments[fragIdx]) - pos
			if availBytes <= 0 {
				continue
			}

			if is16Bit {
				availChars := availBytes / 2
				if availChars == 0 {
					// Odd trailing byte safety
					_, _ = readByte()
					continue
				}
				take := remChars
				if take > availChars {
					take = availChars
				}
				chunk := fragments[fragIdx][pos : pos+take*2]
				sb.WriteString(decodeUTF16LE(chunk))
				pos += take * 2
				remChars -= take
			} else {
				take := remChars
				if take > availBytes {
					take = availBytes
				}
				chunk := fragments[fragIdx][pos : pos+take]
				sb.WriteString(decodeSingleByteChunk(chunk, codepage))
				pos += take
				remChars -= take
			}
		}

		result = append(result, sb.String())

		if richRuns > 0 {
			if !skipBytes(int(richRuns) * 4) {
				break
			}
		}
		if extSize > 0 {
			if !skipBytes(int(extSize)) {
				break
			}
		}
	}

	return result
}

func decodeBIFFCellString(b []byte, biffVersion uint16, codepage uint16) string {
	if len(b) < 2 {
		return ""
	}
	charCount := int(binary.LittleEndian.Uint16(b[0:2]))
	if charCount == 0 {
		return ""
	}

	// BIFF5 / BIFF7: 2-byte byte length followed directly by single-byte characters
	if biffVersion < 0x0600 {
		end := 2 + charCount
		if end > len(b) {
			end = len(b)
		}
		return decodeSingleByteChunk(b[2:end], codepage)
	}

	// BIFF8: 2-byte charCount + 1-byte flags
	if len(b) < 3 {
		return ""
	}
	flags := b[2]
	is16Bit := (flags & 0x01) != 0
	hasExt := (flags & 0x04) != 0
	hasRich := (flags & 0x08) != 0

	offset := 3
	if hasRich && offset+2 <= len(b) {
		offset += 2
	}
	if hasExt && offset+4 <= len(b) {
		offset += 4
	}
	if offset >= len(b) {
		return ""
	}

	if is16Bit {
		byteLen := charCount * 2
		if offset+byteLen > len(b) {
			byteLen = len(b) - offset
		}
		return decodeUTF16LE(b[offset : offset+byteLen])
	}

	end := offset + charCount
	if end > len(b) {
		end = len(b)
	}
	return decodeSingleByteChunk(b[offset:end], codepage)
}

func decodeUTF16LE(b []byte) string {
	n := len(b) / 2
	if n == 0 {
		return ""
	}
	u16 := make([]uint16, n)
	for i := 0; i < n; i++ {
		u16[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	return string(utf16.Decode(u16))
}

func decodeSingleByteChunk(b []byte, codepage uint16) string {
	// Windows-1254 (Turkish) is identical to ISO-8859-1 / Latin-1 for all ASCII and Ç, Ö, Ü, ç, ö, ü,
	// while mapping 0xD0/0xF0 -> Ğ/ğ, 0xDD/0xFD -> İ/ı, 0xDE/0xFE -> Ş/ş.
	switch codepage {
	case 1252:
		if decoded, err := charmap.Windows1252.NewDecoder().Bytes(b); err == nil {
			return string(decoded)
		}
	default:
		if decoded, err := charmap.Windows1254.NewDecoder().Bytes(b); err == nil {
			return string(decoded)
		}
	}
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}

func formatRKNumber(rk uint32) string {
	isMul100 := (rk & 0x01) != 0
	isInt := (rk & 0x02) != 0

	var val float64
	if isInt {
		intVal := int32(rk) >> 2
		val = float64(intVal)
	} else {
		bits := uint64(rk&^uint32(3)) << 34
		val = math.Float64frombits(bits)
	}
	if isMul100 {
		val /= 100.0
	}
	return strconv.FormatFloat(val, 'f', -1, 64)
}

// ============================================================================
// HTML / XML SPREADSHEET (.xls) FALLBACK PARSER
// ============================================================================

func looksLikeHTMLOrXML(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	// Strip UTF-8 BOM if present
	trimmed = bytes.TrimPrefix(trimmed, []byte{0xEF, 0xBB, 0xBF})
	trimmed = bytes.TrimSpace(trimmed)
	return len(trimmed) > 0 && trimmed[0] == '<'
}

func readHTMLOrXMLTableRows(data []byte) ([][]string, error) {
	trimmed := bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	doc, err := html.Parse(bytes.NewReader(trimmed))
	if err != nil {
		return nil, err
	}

	var rows [][]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "tr" || strings.EqualFold(n.Data, "row")) {
			var row []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th" || strings.EqualFold(c.Data, "cell") || strings.EqualFold(c.Data, "data")) {
					row = append(row, extractNodeText(c))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return rows, nil
}

func extractNodeText(n *html.Node) string {
	var sb strings.Builder
	var collect func(*html.Node)
	collect = func(cur *html.Node) {
		if cur.Type == html.TextNode {
			sb.WriteString(cur.Data)
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(n)
	return cleanWhitespace(sb.String())
}

// ============================================================================
// DYNAMIC HEADER SCANNER & TURKISH NORMALIZATION HELPERS
// ============================================================================

func detectGSBHeaderColumns(rows [][]string, firstNonEmptyIdx int) gsbHeaderColumns {
	res := gsbHeaderColumns{
		Found:           false,
		HeaderRowIdx:    firstNonEmptyIdx,
		SchoolCol:       0,
		CityCol:         -1,
		DistrictCol:     -1,
		SeasonCol:       -1,
		SportCol:        -1,
		CategoryTypeCol: -1,
		GenderCol:       -1,
	}

	// Pass 1: Scan rows from the top until a cell matching "Okul Adı" (or "Okul", "School Name", "Kurum Adı") is found
	for rIdx := firstNonEmptyIdx; rIdx < len(rows); rIdx++ {
		row := rows[rIdx]
		schoolIdx := -1
		for cIdx, cell := range row {
			norm := normalizeHeaderKey(cell)
			if norm == "" {
				continue
			}
			if isPrimarySchoolHeader(norm) {
				schoolIdx = cIdx
				break
			}
		}

		if schoolIdx != -1 {
			res.Found = true
			res.HeaderRowIdx = rIdx
			res.SchoolCol = schoolIdx

			for cIdx, cell := range row {
				norm := normalizeHeaderKey(cell)
				if norm == "" {
					continue
				}
				switch {
				case norm == "ilce" || norm == "district" || strings.Contains(norm, "ilce"):
					res.DistrictCol = cIdx
				case norm == "il" || norm == "sehir" || norm == "city" || norm == "province":
					res.CityCol = cIdx
				case norm == "sezon" || norm == "season":
					res.SeasonCol = cIdx
				case norm == "spor dali" || norm == "brans" || norm == "sport" || strings.Contains(norm, "spor dali"):
					res.SportCol = cIdx
				case norm == "kategori turu" || norm == "kategori" || norm == "category" || strings.Contains(norm, "kategori"):
					res.CategoryTypeCol = cIdx
				case norm == "cinsiyet" || norm == "gender":
					res.GenderCol = cIdx
				}
			}
			return res
		}
	}

	return res
}

func isPrimarySchoolHeader(norm string) bool {
	if norm == "" {
		return false
	}
	// Do not confuse Ministry title banners with column headers
	if strings.Contains(norm, "taahhut") || strings.Contains(norm, "bakanlig") || strings.Contains(norm, "sporlari") || strings.Contains(norm, "sistem") {
		return false
	}
	exactMatches := []string{
		"okul adi",
		"okul",
		"school name",
		"school",
		"kurum adi",
		"kurum",
		"takim adi",
		"okul / takim adi",
		"okul/takim adi",
	}
	for _, kw := range exactMatches {
		if norm == kw {
			return true
		}
	}
	return strings.Contains(norm, "okul adi") || strings.Contains(norm, "school name") || strings.Contains(norm, "kurum adi")
}

func isMinistryBannerRow(val string) bool {
	norm := normalizeHeaderKey(val)
	if norm == "" {
		return true
	}
	banners := []string{
		"genclik ve spor bakanligi",
		"okul taahhut listesi",
		"okul sporlari bilgi",
		"rapor tarihi",
	}
	for _, b := range banners {
		if strings.Contains(norm, b) {
			return true
		}
	}
	return false
}

func isRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func cleanWhitespace(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	return strings.Join(fields, " ")
}

var turkishASCIIRep = strings.NewReplacer(
	"İ", "i",
	"I", "i",
	"ı", "i",
	"Ğ", "g",
	"ğ", "g",
	"Ü", "u",
	"ü", "u",
	"Ş", "s",
	"ş", "s",
	"Ö", "o",
	"ö", "o",
	"Ç", "c",
	"ç", "c",
)

func normalizeHeaderKey(s string) string {
	cleaned := cleanWhitespace(s)
	if cleaned == "" {
		return ""
	}
	folded := turkishASCIIRep.Replace(cleaned)
	return strings.ToLower(folded)
}

func normalizeSchoolIdentity(s string) string {
	return normalizeHeaderKey(s)
}

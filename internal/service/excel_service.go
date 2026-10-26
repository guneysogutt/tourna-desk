package service

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"

	"tourna-desk/internal/i18n"
	"tourna-desk/internal/models"
)

func (s *TournamentService) ExportCategoryExcel(w io.Writer, detail *models.CategoryDetail, lang i18n.Lang) error {
	dict := i18n.Get(lang)
	f := excelize.NewFile()
	defer f.Close()

	standingsSheet := dict.SheetStandings
	fixturesSheet := dict.SheetFixtures

	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, standingsSheet); err != nil {
		return err
	}
	if _, err := f.NewSheet(fixturesSheet); err != nil {
		return err
	}

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   14,
			Color:  "#0F172A",
			Family: "Calibri",
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	})

	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Italic: true,
			Size:   10,
			Color:  "#475569",
			Family: "Calibri",
		},
	})

	stageBannerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "#1E293B",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E2E8F0"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	})

	thinBorders := []excelize.Border{
		{Type: "left", Color: "#CBD5E1", Style: 1},
		{Type: "top", Color: "#CBD5E1", Style: 1},
		{Type: "bottom", Color: "#CBD5E1", Style: 1},
		{Type: "right", Color: "#CBD5E1", Style: 1},
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "#FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#1E3A8A"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	rowEvenStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Color:  "#0F172A",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFFFF"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	rowEvenLeftStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   10,
			Color:  "#0F172A",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFFFF"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	rowOddStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Color:  "#0F172A",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F8FAFC"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	rowOddLeftStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   10,
			Color:  "#0F172A",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F8FAFC"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	pointsEvenStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "#1D4ED8",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#EFF6FF"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	pointsOddStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "#1D4ED8",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#DBEAFE"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	// =========================================================================
	// SHEET 1: STANDINGS ("Puan Durumu" / "Standings")
	// =========================================================================
	isVolleyball := detail.Category.SportSlug == models.SportVolleyball

	titleText := fmt.Sprintf("%s - %s (%s)", detail.TournamentName, detail.Category.Name, dict.SportName(detail.Category.SportSlug))
	_ = f.SetCellValue(standingsSheet, "A1", titleText)
	_ = f.SetCellStyle(standingsSheet, "A1", "J1", titleStyle)
	_ = f.SetRowHeight(standingsSheet, 1, 26)

	subText := fmt.Sprintf("%s: %s", dict.LabelGenerated, time.Now().Format("2006-01-02 15:04"))
	_ = f.SetCellValue(standingsSheet, "A2", subText)
	_ = f.SetCellStyle(standingsSheet, "A2", "J2", subtitleStyle)

	standingsHeaders := []string{
		dict.ColRank,
		dict.ColTeam,
		dict.ColPlayed,
		dict.ColWon,
		dict.ColDrawn,
		dict.ColLost,
		dict.ColPoints,
		dict.ColFor,
		dict.ColAgainst,
		dict.ColDiff,
	}
	if isVolleyball {
		standingsHeaders = append(standingsHeaders, dict.ColSetRatio)
	}

	currentRow := 4
	for _, stg := range detail.Stages {
		stgStandings := detail.Standings[stg.ID]

		endCol, _ := excelize.ColumnNumberToName(len(standingsHeaders))
		_ = f.SetCellValue(standingsSheet, fmt.Sprintf("A%d", currentRow), stg.Name)
		_ = f.MergeCell(standingsSheet, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("%s%d", endCol, currentRow))
		_ = f.SetCellStyle(standingsSheet, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("%s%d", endCol, currentRow), stageBannerStyle)
		_ = f.SetRowHeight(standingsSheet, currentRow, 22)
		currentRow++

		for colIdx, h := range standingsHeaders {
			colName, _ := excelize.ColumnNumberToName(colIdx + 1)
			cell := fmt.Sprintf("%s%d", colName, currentRow)
			_ = f.SetCellValue(standingsSheet, cell, h)
			_ = f.SetCellStyle(standingsSheet, cell, cell, headerStyle)
		}
		_ = f.SetRowHeight(standingsSheet, currentRow, 24)
		currentRow++

		for rowIdx, st := range stgStandings {
			isOdd := rowIdx%2 == 1
			cStyle := rowEvenStyle
			lStyle := rowEvenLeftStyle
			pStyle := pointsEvenStyle
			if isOdd {
				cStyle = rowOddStyle
				lStyle = rowOddLeftStyle
				pStyle = pointsOddStyle
			}

			values := []interface{}{
				st.Rank,
				st.TeamName,
				st.Played,
				st.Won,
				st.Drawn,
				st.Lost,
				st.Points,
				st.ScoreFor,
				st.ScoreAgainst,
				st.ScoreDiff,
			}
			if isVolleyball {
				values = append(values, st.SetRatio)
			}

			for colIdx, val := range values {
				colName, _ := excelize.ColumnNumberToName(colIdx + 1)
				cell := fmt.Sprintf("%s%d", colName, currentRow)
				_ = f.SetCellValue(standingsSheet, cell, val)
				if colIdx == 1 {
					_ = f.SetCellStyle(standingsSheet, cell, cell, lStyle)
				} else if colIdx == 6 {
					_ = f.SetCellStyle(standingsSheet, cell, cell, pStyle)
				} else {
					_ = f.SetCellStyle(standingsSheet, cell, cell, cStyle)
				}
			}
			_ = f.SetRowHeight(standingsSheet, currentRow, 20)
			currentRow++
		}

		currentRow++
	}

	_ = f.SetColWidth(standingsSheet, "A", "A", 8)
	_ = f.SetColWidth(standingsSheet, "B", "B", 32)
	_ = f.SetColWidth(standingsSheet, "C", "K", 10)

	// =========================================================================
	// SHEET 2: FIXTURES ("Fikstür" / "Fixtures")
	// =========================================================================
	_ = f.SetCellValue(fixturesSheet, "A1", titleText)
	_ = f.SetCellStyle(fixturesSheet, "A1", "J1", titleStyle)
	_ = f.SetRowHeight(fixturesSheet, 1, 26)

	_ = f.SetCellValue(fixturesSheet, "A2", subText)
	_ = f.SetCellStyle(fixturesSheet, "A2", "J2", subtitleStyle)

	fixtureHeaders := []string{
		dict.ColStage,
		dict.ColRound,
		dict.ColMatchNo,
		dict.ColDate,
		dict.ColVenue,
		dict.ColCourt,
		dict.ColHomeTeam,
		dict.ColScore,
		dict.ColAwayTeam,
		dict.ColStatus,
	}

	fRow := 4
	for colIdx, h := range fixtureHeaders {
		colName, _ := excelize.ColumnNumberToName(colIdx + 1)
		cell := fmt.Sprintf("%s%d", colName, fRow)
		_ = f.SetCellValue(fixturesSheet, cell, h)
		_ = f.SetCellStyle(fixturesSheet, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(fixturesSheet, fRow, 24)
	fRow++

	for idx, m := range detail.Matches {
		isOdd := idx%2 == 1
		cStyle := rowEvenStyle
		lStyle := rowEvenLeftStyle
		pStyle := pointsEvenStyle
		if isOdd {
			cStyle = rowOddStyle
			lStyle = rowOddLeftStyle
			pStyle = pointsOddStyle
		}

		scoreStr := "-"
		if m.IsBye {
			scoreStr = dict.LabelBye
		} else if m.Status == models.StatusCompleted {
			if m.HomePenalties != nil && m.AwayPenalties != nil {
				scoreStr = fmt.Sprintf("%d - %d (P: %d-%d)", m.HomeScore, m.AwayScore, *m.HomePenalties, *m.AwayPenalties)
			} else {
				scoreStr = fmt.Sprintf("%d - %d", m.HomeScore, m.AwayScore)
			}
		}

		roundStr := fmt.Sprintf("%s %d", dict.LabelRound, m.Round)
		statusStr := dict.FormatStatus(m.Status, m.IsBye)
		if m.IsForfeit {
			if lang == i18n.LangEN {
				statusStr = fmt.Sprintf("Forfeit (%d-%d)", m.HomeScore, m.AwayScore)
			} else {
				statusStr = fmt.Sprintf("Hükmen (%d-%d)", m.HomeScore, m.AwayScore)
			}
		}

		rowVals := []interface{}{
			m.StageName,
			roundStr,
			m.MatchNumber,
			m.ScheduledAt,
			m.Venue,
			m.Court,
			m.HomeTeamName,
			scoreStr,
			m.AwayTeamName,
			statusStr,
		}

		for colIdx, val := range rowVals {
			colName, _ := excelize.ColumnNumberToName(colIdx + 1)
			cell := fmt.Sprintf("%s%d", colName, fRow)
			_ = f.SetCellValue(fixturesSheet, cell, val)
			if colIdx == 6 || colIdx == 8 {
				_ = f.SetCellStyle(fixturesSheet, cell, cell, lStyle)
			} else if colIdx == 7 {
				_ = f.SetCellStyle(fixturesSheet, cell, cell, pStyle)
			} else {
				_ = f.SetCellStyle(fixturesSheet, cell, cell, cStyle)
			}
		}
		_ = f.SetRowHeight(fixturesSheet, fRow, 20)
		fRow++
	}

	_ = f.SetColWidth(fixturesSheet, "A", "B", 14)
	_ = f.SetColWidth(fixturesSheet, "C", "C", 10)
	_ = f.SetColWidth(fixturesSheet, "D", "F", 18)
	_ = f.SetColWidth(fixturesSheet, "G", "G", 28)
	_ = f.SetColWidth(fixturesSheet, "H", "H", 12)
	_ = f.SetColWidth(fixturesSheet, "I", "I", 28)
	_ = f.SetColWidth(fixturesSheet, "J", "J", 16)

	f.SetActiveSheet(0)
	return f.Write(w)
}

// ExportTeamTemplateExcel generates a clean sample Excel file ("Okul Adı")
func (s *TournamentService) ExportTeamTemplateExcel(w io.Writer, lang i18n.Lang) error {
	dict := i18n.Get(lang)
	f := excelize.NewFile()
	defer f.Close()

	sheetName := dict.TemplateSheetName
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return err
	}

	thinBorders := []excelize.Border{
		{Type: "left", Color: "#CBD5E1", Style: 1},
		{Type: "top", Color: "#CBD5E1", Style: 1},
		{Type: "bottom", Color: "#CBD5E1", Style: 1},
		{Type: "right", Color: "#CBD5E1", Style: 1},
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "#FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#1E3A8A"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	rowStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Color:  "#0F172A",
			Family: "Calibri",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Border: thinBorders,
	})

	_ = f.SetCellValue(sheetName, "A1", dict.TemplateColSchool)
	_ = f.SetCellStyle(sheetName, "A1", "A1", headerStyle)
	_ = f.SetRowHeight(sheetName, 1, 24)

	sampleSchools := []string{
		"Atatürk Anadolu Lisesi",
		"Cumhuriyet Fen Lisesi",
		"Mimar Sinan Ortaokulu",
	}

	for idx, school := range sampleSchools {
		rNum := idx + 2
		_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", rNum), school)
		_ = f.SetCellStyle(sheetName, fmt.Sprintf("A%d", rNum), fmt.Sprintf("A%d", rNum), rowStyle)
		_ = f.SetRowHeight(sheetName, rNum, 20)
	}

	_ = f.SetColWidth(sheetName, "A", "A", 42)

	return f.Write(w)
}

func knockoutRoundTitleI18n(matchesInRound int, roundNum int, lang i18n.Lang) string {
	isEN := lang == i18n.LangEN
	switch matchesInRound {
	case 32:
		if isEN {
			return "Round of 64"
		}
		return "Son 64"
	case 16:
		if isEN {
			return "Round of 32"
		}
		return "Son 32"
	case 8:
		if isEN {
			return "Round of 16"
		}
		return "Son 16"
	case 4:
		if isEN {
			return "Quarter-Finals"
		}
		return "Çeyrek Final"
	case 2:
		if isEN {
			return "Semi-Finals"
		}
		return "Yarı Final"
	case 1:
		return "Final"
	default:
		if isEN {
			return fmt.Sprintf("Round %d", roundNum)
		}
		return fmt.Sprintf("%d. Tur", roundNum)
	}
}

// BuildKnockoutExcel constructs an Excel workbook visually representing a single-elimination bracket
// with rounds placed side-by-side in columns (Col A-B: Round 1, Col D-E: Round 2, Col G-H: Final & 3rd Place, etc.).
func BuildKnockoutExcel(detail *models.CategoryDetail, lang i18n.Lang) (*excelize.File, error) {
	dict := i18n.Get(lang)
	isEN := lang == i18n.LangEN

	f := excelize.NewFile()
	sheetName := "Eleme Ağacı"
	if isEN {
		sheetName = "Knockout Bracket"
	}
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, err
	}

	thinBorders := []excelize.Border{
		{Type: "left", Color: "#CBD5E1", Style: 1},
		{Type: "top", Color: "#CBD5E1", Style: 1},
		{Type: "bottom", Color: "#CBD5E1", Style: 1},
		{Type: "right", Color: "#CBD5E1", Style: 1},
	}

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "#0F172A", Family: "Calibri"},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})

	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 10, Color: "#475569", Family: "Calibri"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1E3A8A"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorders,
	})

	thirdPlaceHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#B45309"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorders,
	})

	teamStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#0F172A", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFFFFF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    thinBorders,
	})

	winnerTeamStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "#065F46", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ECFDF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    thinBorders,
	})

	byeTeamStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 10, Color: "#92400E", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFFBEB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    thinBorders,
	})

	scoreStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#334155", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F8FAFC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorders,
	})

	winnerScoreStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "#065F46", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D1FAE5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorders,
	})

	byeBadgeStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "#B45309", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FEF3C7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorders,
	})

	titleText := fmt.Sprintf("%s - %s (%s)", detail.TournamentName, detail.Category.Name, dict.SportName(detail.Category.SportSlug))
	_ = f.SetCellValue(sheetName, "A1", titleText)
	_ = f.SetCellStyle(sheetName, "A1", "H1", titleStyle)
	_ = f.SetRowHeight(sheetName, 1, 26)

	subText := fmt.Sprintf("%s: %s", dict.LabelGenerated, time.Now().Format("2006-01-02 15:04"))
	_ = f.SetCellValue(sheetName, "A2", subText)
	_ = f.SetCellStyle(sheetName, "A2", "H2", subtitleStyle)

	// Locate the active knockout (or final_four) stage
	var bracketStageID int64
	for _, stg := range detail.Stages {
		if stg.StageType == "knockout" {
			bracketStageID = stg.ID
		}
	}
	if bracketStageID == 0 {
		for _, stg := range detail.Stages {
			if stg.StageType == "final_four" {
				bracketStageID = stg.ID
			}
		}
	}

	var bracketMatches []models.Match
	for _, m := range detail.Matches {
		if bracketStageID == 0 || m.StageID == bracketStageID {
			bracketMatches = append(bracketMatches, m)
		}
	}

	// Group matches by round (separating 3rd place match)
	roundsMap := make(map[int][]models.Match)
	var roundNums []int
	var thirdPlaceMatch *models.Match

	for i := range bracketMatches {
		m := bracketMatches[i]
		if m.BracketSlot == string(models.BracketSlotThirdPlace) {
			mCopy := m
			thirdPlaceMatch = &mCopy
			continue
		}
		if _, exists := roundsMap[m.Round]; !exists {
			roundNums = append(roundNums, m.Round)
		}
		roundsMap[m.Round] = append(roundsMap[m.Round], m)
	}
	sort.Ints(roundNums)

	baseRow := 6
	maxRowUsed := baseRow + 4

	writeMatchNode := func(teamColIdx, scoreColIdx, homeRow int, m models.Match) {
		awayRow := homeRow + 1
		teamCol, _ := excelize.ColumnNumberToName(teamColIdx)
		scoreCol, _ := excelize.ColumnNumberToName(scoreColIdx)

		homeTeamCell := fmt.Sprintf("%s%d", teamCol, homeRow)
		homeScoreCell := fmt.Sprintf("%s%d", scoreCol, homeRow)
		awayTeamCell := fmt.Sprintf("%s%d", teamCol, awayRow)
		awayScoreCell := fmt.Sprintf("%s%d", scoreCol, awayRow)

		isBye := m.IsBye || m.Status == models.StatusByeAdvanced
		isCompleted := m.Status == models.StatusCompleted
		hasPen := m.HomePenalties != nil && m.AwayPenalties != nil

		homeWon := isBye || (isCompleted && (m.HomeScore > m.AwayScore || (m.HomeScore == m.AwayScore && hasPen && *m.HomePenalties > *m.AwayPenalties)))
		awayWon := !isBye && isCompleted && (m.AwayScore > m.HomeScore || (m.HomeScore == m.AwayScore && hasPen && *m.AwayPenalties > *m.HomePenalties))

		byeLabel := dict.LabelBye
		if isBye {
			// Clear BYE line representation: e.g. "[Okul Adı] - BAY"
			_ = f.SetCellValue(sheetName, homeTeamCell, fmt.Sprintf("%s - %s", m.HomeTeamName, byeLabel))
			_ = f.SetCellValue(sheetName, homeScoreCell, byeLabel)
			_ = f.SetCellStyle(sheetName, homeTeamCell, homeTeamCell, winnerTeamStyle)
			_ = f.SetCellStyle(sheetName, homeScoreCell, homeScoreCell, byeBadgeStyle)

			_ = f.SetCellValue(sheetName, awayTeamCell, byeLabel)
			_ = f.SetCellValue(sheetName, awayScoreCell, "-")
			_ = f.SetCellStyle(sheetName, awayTeamCell, awayTeamCell, byeTeamStyle)
			_ = f.SetCellStyle(sheetName, awayScoreCell, awayScoreCell, byeBadgeStyle)
		} else {
			_ = f.SetCellValue(sheetName, homeTeamCell, m.HomeTeamName)
			_ = f.SetCellValue(sheetName, awayTeamCell, m.AwayTeamName)

			if isCompleted {
				if hasPen {
					_ = f.SetCellValue(sheetName, homeScoreCell, fmt.Sprintf("%d (P:%d)", m.HomeScore, *m.HomePenalties))
					_ = f.SetCellValue(sheetName, awayScoreCell, fmt.Sprintf("%d (P:%d)", m.AwayScore, *m.AwayPenalties))
				} else {
					_ = f.SetCellValue(sheetName, homeScoreCell, m.HomeScore)
					_ = f.SetCellValue(sheetName, awayScoreCell, m.AwayScore)
				}
			} else {
				_ = f.SetCellValue(sheetName, homeScoreCell, "-")
				_ = f.SetCellValue(sheetName, awayScoreCell, "-")
			}

			if homeWon {
				_ = f.SetCellStyle(sheetName, homeTeamCell, homeTeamCell, winnerTeamStyle)
				_ = f.SetCellStyle(sheetName, homeScoreCell, homeScoreCell, winnerScoreStyle)
			} else {
				_ = f.SetCellStyle(sheetName, homeTeamCell, homeTeamCell, teamStyle)
				_ = f.SetCellStyle(sheetName, homeScoreCell, homeScoreCell, scoreStyle)
			}

			if awayWon {
				_ = f.SetCellStyle(sheetName, awayTeamCell, awayTeamCell, winnerTeamStyle)
				_ = f.SetCellStyle(sheetName, awayScoreCell, awayScoreCell, winnerScoreStyle)
			} else {
				_ = f.SetCellStyle(sheetName, awayTeamCell, awayTeamCell, teamStyle)
				_ = f.SetCellStyle(sheetName, awayScoreCell, awayScoreCell, scoreStyle)
			}
		}

		_ = f.SetRowHeight(sheetName, homeRow, 20)
		_ = f.SetRowHeight(sheetName, awayRow, 20)
	}

	for rIdx, rNum := range roundNums {
		rMatches := roundsMap[rNum]
		sort.Slice(rMatches, func(i, j int) bool {
			return rMatches[i].MatchNumber < rMatches[j].MatchNumber
		})

		teamColIdx := rIdx*3 + 1  // Col A (1), Col D (4), Col G (7), Col J (10)...
		scoreColIdx := rIdx*3 + 2 // Col B (2), Col E (5), Col H (8), Col K (11)...
		spacerColIdx := rIdx*3 + 3

		teamCol, _ := excelize.ColumnNumberToName(teamColIdx)
		scoreCol, _ := excelize.ColumnNumberToName(scoreColIdx)
		spacerCol, _ := excelize.ColumnNumberToName(spacerColIdx)

		_ = f.SetColWidth(sheetName, teamCol, teamCol, 30)
		_ = f.SetColWidth(sheetName, scoreCol, scoreCol, 12)
		_ = f.SetColWidth(sheetName, spacerCol, spacerCol, 5)

		// Round Header at Row 4
		headerTitle := knockoutRoundTitleI18n(len(rMatches), rNum, lang)
		hStart := fmt.Sprintf("%s4", teamCol)
		hEnd := fmt.Sprintf("%s4", scoreCol)
		_ = f.SetCellValue(sheetName, hStart, headerTitle)
		_ = f.MergeCell(sheetName, hStart, hEnd)
		_ = f.SetCellStyle(sheetName, hStart, hEnd, headerStyle)
		_ = f.SetRowHeight(sheetName, 4, 24)

		step := 4 * (1 << rIdx)
		offset := (1 << (rIdx + 1)) - 2

		for mIdx, m := range rMatches {
			homeRow := baseRow + offset + mIdx*step
			writeMatchNode(teamColIdx, scoreColIdx, homeRow, m)

			// Merge vertical connector spacing cells between paired matches
			if mIdx%2 == 0 && mIdx+1 < len(rMatches) {
				nextHomeRow := baseRow + offset + (mIdx+1)*step
				if nextHomeRow-1 > homeRow+2 {
					connStart := fmt.Sprintf("%s%d", spacerCol, homeRow+1)
					connEnd := fmt.Sprintf("%s%d", spacerCol, nextHomeRow)
					_ = f.MergeCell(sheetName, connStart, connEnd)
				}
			}

			if homeRow+2 > maxRowUsed {
				maxRowUsed = homeRow + 2
			}
		}

		// Place 3rd Place Match below the Final in the last round column
		if rIdx == len(roundNums)-1 && thirdPlaceMatch != nil {
			tpHeaderRow := maxRowUsed + 2
			if len(rMatches) == 1 {
				finalRow := baseRow + offset
				if finalRow+4 > tpHeaderRow {
					tpHeaderRow = finalRow + 4
				}
			}
			tpTitle := "3.'lük Maçı"
			if isEN {
				tpTitle = "3rd Place Match"
			}
			tpStart := fmt.Sprintf("%s%d", teamCol, tpHeaderRow)
			tpEnd := fmt.Sprintf("%s%d", scoreCol, tpHeaderRow)
			_ = f.SetCellValue(sheetName, tpStart, tpTitle)
			_ = f.MergeCell(sheetName, tpStart, tpEnd)
			_ = f.SetCellStyle(sheetName, tpStart, tpEnd, thirdPlaceHeaderStyle)
			_ = f.SetRowHeight(sheetName, tpHeaderRow, 22)

			writeMatchNode(teamColIdx, scoreColIdx, tpHeaderRow+1, *thirdPlaceMatch)
		}
	}

	return f, nil
}

// ExportKnockoutBracketExcel writes the visual knockout bracket Excel file to w.
func (s *TournamentService) ExportKnockoutBracketExcel(w io.Writer, detail *models.CategoryDetail, lang i18n.Lang) error {
	f, err := BuildKnockoutExcel(detail, lang)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Write(w)
}

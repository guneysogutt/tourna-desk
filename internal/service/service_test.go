package service_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"tourna-desk/internal/database"
	"tourna-desk/internal/i18n"
	"tourna-desk/internal/models"
	"tourna-desk/internal/service"
)

func TestExcelTeamTemplateAndBulkImport(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test_tourna.db")
	db, err := database.Open(tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	svc := service.NewTournamentService(db)

	cat, err := svc.CreateCategory(1, models.SportBasketball, "Genç Erkekler Basketbol Test", "Grup A")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// 1. Generate the Excel template
	var buf bytes.Buffer
	if err := svc.ExportTeamTemplateExcel(&buf, i18n.LangTR); err != nil {
		t.Fatalf("failed to export team template: %v", err)
	}

	// 2. Import teams directly from the generated template buffer
	imported, err := svc.ImportTeamsFromExcel(cat.ID, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("failed to import teams from excel: %v", err)
	}
	if len(imported) != 3 {
		t.Fatalf("expected 3 sample teams imported from template, got %d", len(imported))
	}

	if imported[0].DisplayName != "Atatürk Anadolu Lisesi" {
		t.Errorf("expected 'Atatürk Anadolu Lisesi', got %q", imported[0].DisplayName)
	}
	if imported[1].DisplayName != "Cumhuriyet Fen Lisesi" {
		t.Errorf("expected 'Cumhuriyet Fen Lisesi', got %q", imported[1].DisplayName)
	}
	if imported[0].StageID != nil {
		t.Errorf("expected imported team stage_id to be nil (decoupled from groups), got %v", imported[0].StageID)
	}
}

func TestMultiRoundTournamentLifecycleEndToEnd(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test_lifecycle.db")
	db, err := database.Open(tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	svc := service.NewTournamentService(db)

	cat, err := svc.CreateCategory(1, models.SportVolleyball, "Genç Kızlar Voleybol", "Grup A")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Register 12 teams
	var teamIDs []int64
	for i := 1; i <= 12; i++ {
		tm, err := svc.CreateTeam(cat.ID, nil, "Okul "+string(rune('A'+i-1)), "")
		if err != nil {
			t.Fatalf("failed to create team %d: %v", i, err)
		}
		teamIDs = append(teamIDs, tm.ID)
	}

	// Select 3 seeded teams and run Stage 1 Initial Draw
	seeded := []int64{teamIDs[0], teamIDs[4], teamIDs[8]}
	detail, err := svc.ExecuteInitialSeededDraw(cat.ID, service.InitialDrawInput{
		SeededTeamIDs:    seeded,
		GenerateFixtures: true,
		DefaultVenue:     "Merkez Spor Salonu",
		DefaultCourt:     "Kort 1",
		StartDate:        "2026-10-20",
	})
	if err != nil {
		t.Fatalf("ExecuteInitialSeededDraw failed: %v", err)
	}

	if len(detail.Stages) != 3 {
		t.Fatalf("expected 3 groups of 4 teams in Round 1, got %d", len(detail.Stages))
	}

	// Complete all Round 1 matches so standings are deterministic
	for _, m := range detail.Matches {
		_, err := svc.UpdateMatch(m.ID, service.UpdateMatchInput{
			HomeScore: 3,
			AwayScore: 1,
			Status:    models.StatusCompleted,
		})
		if err != nil {
			t.Fatalf("UpdateMatch failed: %v", err)
		}
	}

	// Prepare Next Round Pots (3 groups -> 3 in Pot 1, 3 in Pot 2)
	detail, err = svc.PrepareCategoryPots(cat.ID, nil)
	if err != nil {
		t.Fatalf("PrepareCategoryPots failed: %v", err)
	}
	if len(detail.PotAssignments) != 6 {
		t.Fatalf("expected 6 qualified teams in pots, got %d", len(detail.PotAssignments))
	}

	// Add 2 wildcard teams via manual pot editing to form 8 teams (4 in Pot 1, 4 in Pot 2)
	inPot := make(map[int64]bool)
	var pot1IDs, pot2IDs []int64
	for _, pa := range detail.PotAssignments {
		inPot[pa.TeamID] = true
		if pa.PotNumber == 1 {
			pot1IDs = append(pot1IDs, pa.TeamID)
		} else {
			pot2IDs = append(pot2IDs, pa.TeamID)
		}
	}
	for _, tid := range teamIDs {
		if !inPot[tid] {
			if len(pot1IDs) < 4 {
				pot1IDs = append(pot1IDs, tid)
			} else if len(pot2IDs) < 4 {
				pot2IDs = append(pot2IDs, tid)
				break
			}
		}
	}

	detail, err = svc.SaveCategoryPots(cat.ID, service.SavePotsInput{
		Pot1TeamIDs: pot1IDs,
		Pot2TeamIDs: pot2IDs,
	})
	if err != nil {
		t.Fatalf("SaveCategoryPots failed: %v", err)
	}
	if len(detail.PotAssignments) != 8 {
		t.Fatalf("expected 8 teams in edited pots, got %d", len(detail.PotAssignments))
	}

	// Execute Next Round Draw -> creates 2 Round 2 groups of 4 teams each
	detail, err = svc.ExecuteCategoryNextRoundDraw(cat.ID, service.NextRoundDrawInput{
		GenerateFixtures: true,
		DefaultVenue:     "Merkez Spor Salonu",
		DefaultCourt:     "Kort 1",
		StartDate:        "2026-10-24",
	})
	if err != nil {
		t.Fatalf("ExecuteCategoryNextRoundDraw failed: %v", err)
	}

	var round2Stages []models.Stage
	for _, st := range detail.Stages {
		if st.RoundNumber == 2 && st.StageType == "group" {
			round2Stages = append(round2Stages, st)
		}
	}
	if len(round2Stages) != 2 {
		t.Fatalf("expected 2 groups in Round 2, got %d", len(round2Stages))
	}

	// Also verify Round 1 stages still have all 4 of their original teams preserved!
	for _, st := range detail.Stages {
		if st.RoundNumber == 1 {
			if len(st.Teams) != 4 {
				t.Fatalf("expected Round 1 stage %s to preserve 4 teams, got %d", st.Name, len(st.Teams))
			}
		}
	}

	// Build Final Four from the 2 Round 2 groups
	detail, err = svc.BuildCategoryFinalFour(cat.ID, service.FinalFourInput{
		GroupAStageID: round2Stages[0].ID,
		GroupBStageID: round2Stages[1].ID,
		DefaultVenue:  "Merkez Arena",
		DefaultCourt:  "Ana Saha",
		StartDate:     "2026-10-28",
	})
	if err != nil {
		t.Fatalf("BuildCategoryFinalFour failed: %v", err)
	}

	var sf1Match, sf2Match *models.Match
	for i := range detail.Matches {
		m := &detail.Matches[i]
		if m.BracketSlot == string(models.BracketSlotSemiFinal1) {
			sf1Match = m
		} else if m.BracketSlot == string(models.BracketSlotSemiFinal2) {
			sf2Match = m
		}
	}
	if sf1Match == nil || sf2Match == nil {
		t.Fatalf("expected Semi-Final 1 and Semi-Final 2 matches to exist")
	}

	// Complete Semi-Final 1 (Home wins 3-1) and Semi-Final 2 (Away wins 2-3)
	_, err = svc.UpdateMatch(sf1Match.ID, service.UpdateMatchInput{
		HomeScore: 3,
		AwayScore: 1,
		Status:    models.StatusCompleted,
	})
	if err != nil {
		t.Fatalf("UpdateMatch SF1 failed: %v", err)
	}
	_, err = svc.UpdateMatch(sf2Match.ID, service.UpdateMatchInput{
		HomeScore: 2,
		AwayScore: 3,
		Status:    models.StatusCompleted,
	})
	if err != nil {
		t.Fatalf("UpdateMatch SF2 failed: %v", err)
	}

	detail, err = svc.GetCategoryDetail(cat.ID)
	if err != nil {
		t.Fatalf("GetCategoryDetail failed: %v", err)
	}

	for _, m := range detail.Matches {
		if m.BracketSlot == string(models.BracketSlotFinal) {
			if m.HomeTeamID == nil || *m.HomeTeamID != *sf1Match.HomeTeamID {
				t.Fatalf("expected Final home team to be SF1 winner (%v), got %v", *sf1Match.HomeTeamID, m.HomeTeamID)
			}
			if m.AwayTeamID == nil || *m.AwayTeamID != *sf2Match.AwayTeamID {
				t.Fatalf("expected Final away team to be SF2 winner (%v), got %v", *sf2Match.AwayTeamID, m.AwayTeamID)
			}
		}
		if m.BracketSlot == string(models.BracketSlotThirdPlace) {
			if m.HomeTeamID == nil || *m.HomeTeamID != *sf1Match.AwayTeamID {
				t.Fatalf("expected 3rd Place home team to be SF1 loser (%v), got %v", *sf1Match.AwayTeamID, m.HomeTeamID)
			}
			if m.AwayTeamID == nil || *m.AwayTeamID != *sf2Match.HomeTeamID {
				t.Fatalf("expected 3rd Place away team to be SF2 loser (%v), got %v", *sf2Match.HomeTeamID, m.AwayTeamID)
			}
		}
	}
}

func TestSlotIndexFixtureKeyAndForfeitStandings(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test_slot_forfeit.db")
	db, err := database.Open(tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	svc := service.NewTournamentService(db)
	cat, err := svc.CreateCategory(1, models.SportBasketball, "Genç Erkekler Basketbol", "Grup A")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	var teamIDs []int64
	for i := 1; i <= 12; i++ {
		tm, err := svc.CreateTeam(cat.ID, nil, "Basket Lisesi "+string(rune('A'+i-1)), "")
		if err != nil {
			t.Fatalf("failed to create team %d: %v", i, err)
		}
		teamIDs = append(teamIDs, tm.ID)
	}

	// Pick highest IDs as seeded teams to verify slot_index=1 overrides ID sorting
	seeded := []int64{teamIDs[9], teamIDs[10], teamIDs[11]}
	detail, err := svc.ExecuteInitialSeededDraw(cat.ID, service.InitialDrawInput{
		SeededTeamIDs:    seeded,
		GenerateFixtures: true,
		DefaultVenue:     "Atatürk Spor Salonu",
		DefaultCourt:     "Salon 1",
		StartDate:        "2026-10-20",
	})
	if err != nil {
		t.Fatalf("ExecuteInitialSeededDraw failed: %v", err)
	}

	for _, st := range detail.Stages {
		if len(st.Teams) != 4 {
			t.Fatalf("expected 4 teams in group %s, got %d", st.Name, len(st.Teams))
		}
		if !st.Teams[0].IsSeeded || st.Teams[0].SlotIndex != 1 {
			t.Fatalf("expected Position 1 in %s to be seeded with slot_index=1, got team=%s isSeeded=%v slot=%d",
				st.Name, st.Teams[0].DisplayName, st.Teams[0].IsSeeded, st.Teams[0].SlotIndex)
		}
	}

	// Mark the first match of Group A as a Forfeit (Hükmen) won by Home Team
	firstMatch := detail.Matches[0]
	winnerID := *firstMatch.HomeTeamID
	loserID := *firstMatch.AwayTeamID
	updated, err := svc.UpdateMatch(firstMatch.ID, service.UpdateMatchInput{
		Status:              models.StatusCompleted,
		IsForfeit:           true,
		ForfeitWinnerTeamID: &winnerID,
	})
	if err != nil {
		t.Fatalf("UpdateMatch forfeit failed: %v", err)
	}
	if !updated.IsForfeit || updated.HomeScore != 20 || updated.AwayScore != 0 {
		t.Fatalf("expected Basketball forfeit auto-score 20-0, got %d-%d (isForfeit=%v)",
			updated.HomeScore, updated.AwayScore, updated.IsForfeit)
	}

	detail, err = svc.GetCategoryDetail(cat.ID)
	if err != nil {
		t.Fatalf("GetCategoryDetail failed: %v", err)
	}

	rows := detail.Standings[firstMatch.StageID]
	for _, r := range rows {
		if r.TeamID == winnerID && r.Points != 2 {
			t.Errorf("expected basketball forfeit winner to have 2 pts, got %d", r.Points)
		}
		if r.TeamID == loserID && r.Points != 0 {
			t.Errorf("expected basketball forfeit loser to receive 0 pts (penalty), got %d", r.Points)
		}
	}
}

func TestKnockoutPenaltiesAndEditableMatchups(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test_knockout_penalties.db")
	db, err := database.Open(tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	svc := service.NewTournamentService(db)
	cat, err := svc.CreateCategory(1, models.SportFutsal, "Genç Erkek Futsal Eleme", "Grup A")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	var teamIDs []int64
	for i := 1; i <= 8; i++ {
		tm, err := svc.CreateTeam(cat.ID, nil, "Futsal Okulu "+string(rune('A'+i-1)))
		if err != nil {
			t.Fatalf("failed to create team %d: %v", i, err)
		}
		teamIDs = append(teamIDs, tm.ID)
	}

	// Build 8-team knockout bracket via InitialDraw with Format="knockout"
	detail, err := svc.ExecuteInitialSeededDraw(cat.ID, service.InitialDrawInput{
		Format:        "knockout",
		SeededTeamIDs: []int64{teamIDs[0], teamIDs[1]},
		DefaultVenue:  "Merkez Salon",
		DefaultCourt:  "Saha 1",
		StartDate:     "2026-10-25",
	})
	if err != nil {
		t.Fatalf("ExecuteInitialSeededDraw(knockout) failed: %v", err)
	}
	if len(detail.Stages) != 1 || detail.Stages[0].StageType != "knockout" {
		t.Fatalf("expected 1 knockout stage, got %+v", detail.Stages)
	}
	if len(detail.Matches) != 8 {
		t.Fatalf("expected 8 matches in 8-team knockout bracket, got %d", len(detail.Matches))
	}

	// Test editable pairings before matches start: swap home teams between Match 1 and Match 2
	m1 := detail.Matches[0]
	m2 := detail.Matches[1]
	origM1Home := *m1.HomeTeamID
	origM2Home := *m2.HomeTeamID

	detail, err = svc.SwapMatchPairings(cat.ID, service.SwapMatchPairingsInput{
		MatchAID: m1.ID,
		SlotA:    "home",
		MatchBID: m2.ID,
		SlotB:    "home",
	})
	if err != nil {
		t.Fatalf("SwapMatchPairings failed: %v", err)
	}
	if *detail.Matches[0].HomeTeamID != origM2Home || *detail.Matches[1].HomeTeamID != origM1Home {
		t.Fatalf("expected swapped home teams %d and %d, got %d and %d",
			origM2Home, origM1Home, *detail.Matches[0].HomeTeamID, *detail.Matches[1].HomeTeamID)
	}

	// Elimination match cannot end in a draw without penalties
	_, err = svc.UpdateMatch(m1.ID, service.UpdateMatchInput{
		HomeScore: 2,
		AwayScore: 2,
		Status:    models.StatusCompleted,
	})
	if err == nil {
		t.Fatalf("expected error when saving a draw without penalties in an elimination match")
	}

	// Elimination match with decisive penalties (2-2, penalties 4-5 -> Away wins) advances Away to Semi-Final 1 Home slot
	hp, ap := 4, 5
	expectedAdvancingTeamID := *detail.Matches[0].AwayTeamID
	_, err = svc.UpdateMatch(m1.ID, service.UpdateMatchInput{
		HomeScore:     2,
		AwayScore:     2,
		HomePenalties: &hp,
		AwayPenalties: &ap,
		Status:        models.StatusCompleted,
	})
	if err != nil {
		t.Fatalf("UpdateMatch with decisive penalties failed: %v", err)
	}

	detail, err = svc.GetCategoryDetail(cat.ID)
	if err != nil {
		t.Fatalf("GetCategoryDetail failed: %v", err)
	}
	// Find Semi-Final 1
	var sf1 *models.Match
	for i := range detail.Matches {
		if detail.Matches[i].BracketSlot == string(models.BracketSlotSemiFinal1) {
			sf1 = &detail.Matches[i]
			break
		}
	}
	if sf1 == nil || sf1.HomeTeamID == nil || *sf1.HomeTeamID != expectedAdvancingTeamID {
		t.Fatalf("expected QF1 penalty winner (%d) to advance to SF1 Home slot, got %+v", expectedAdvancingTeamID, sf1)
	}

	// Also test 7-team dynamic BYE knockout bracket + Excel export
	cat7, err := svc.CreateCategory(1, models.SportBasketball, "Yıldız Erkekler Eleme (7 Takım)", "Eleme")
	if err != nil {
		t.Fatalf("CreateCategory(cat7) failed: %v", err)
	}
	for i := 1; i <= 7; i++ {
		if _, err := svc.CreateTeam(cat7.ID, nil, fmt.Sprintf("Anadolu Lisesi %d", i)); err != nil {
			t.Fatalf("CreateTeam(%d) failed: %v", i, err)
		}
	}
	detail7, err := svc.ExecuteInitialSeededDraw(cat7.ID, service.InitialDrawInput{
		Format:       "knockout",
		DefaultVenue: "Merkez Salon",
		DefaultCourt: "Kort A",
		StartDate:    "2026-10-26",
	})
	if err != nil {
		t.Fatalf("ExecuteInitialSeededDraw(7 teams knockout) failed: %v", err)
	}
	if len(detail7.Matches) != 8 {
		t.Fatalf("expected 8 bracket nodes for 7 teams (S=8, B=1), got %d", len(detail7.Matches))
	}
	var byeMatches int
	for _, m := range detail7.Matches {
		if m.IsBye && m.Status == models.StatusByeAdvanced {
			byeMatches++
			if m.NextMatchID == nil {
				t.Fatalf("expected NextMatchID to be linked on Round 1 BYE match")
			}
		}
	}
	if byeMatches != 1 {
		t.Fatalf("expected 1 BYE match for 7 teams, got %d", byeMatches)
	}

	var excelBuf bytes.Buffer
	if err := svc.ExportKnockoutBracketExcel(&excelBuf, detail7, "tr"); err != nil {
		t.Fatalf("ExportKnockoutBracketExcel failed: %v", err)
	}
	if excelBuf.Len() == 0 {
		t.Fatalf("expected non-empty Excel buffer from ExportKnockoutBracketExcel")
	}
	if detail7.Category.Format != "knockout" || detail7.Category.Status != "ACTIVE" {
		t.Fatalf("expected category format='knockout' and status='ACTIVE', got format=%q status=%q", detail7.Category.Format, detail7.Category.Status)
	}

	// Test ResetCategoryFixtures: clears matches & stages while preserving all 7 registered teams and resetting status to AWAITING_DRAW
	resetDetail, err := svc.ResetCategoryFixtures(cat7.ID, nil)
	if err != nil {
		t.Fatalf("ResetCategoryFixtures failed: %v", err)
	}
	if len(resetDetail.Matches) != 0 || len(resetDetail.Stages) != 0 {
		t.Fatalf("expected 0 matches and 0 stages after reset, got %d matches and %d stages", len(resetDetail.Matches), len(resetDetail.Stages))
	}
	if len(resetDetail.Teams) != 7 {
		t.Fatalf("expected all 7 registered teams to be preserved after reset, got %d", len(resetDetail.Teams))
	}
	if resetDetail.Category.Status != "AWAITING_DRAW" {
		t.Fatalf("expected category status='AWAITING_DRAW' after reset, got %q", resetDetail.Category.Status)
	}
}

func buildTestBIFF8XLS(rows [][]string) []byte {
	var sstStrings []string
	sstIndex := make(map[string]uint32)
	totalRefs := uint32(0)

	for _, r := range rows {
		for _, cell := range r {
			if cell == "" {
				continue
			}
			totalRefs++
			if _, ok := sstIndex[cell]; !ok {
				sstIndex[cell] = uint32(len(sstStrings))
				sstStrings = append(sstStrings, cell)
			}
		}
	}

	var wb bytes.Buffer
	writeRec := func(id uint16, payload []byte) {
		var hdr [4]byte
		binary.LittleEndian.PutUint16(hdr[0:2], id)
		binary.LittleEndian.PutUint16(hdr[2:4], uint16(len(payload)))
		wb.Write(hdr[:])
		wb.Write(payload)
	}

	// 1. Workbook Globals BOF (0x0809, version=0x0600, type=0x0005)
	bofGlobals := make([]byte, 16)
	binary.LittleEndian.PutUint16(bofGlobals[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bofGlobals[2:4], 0x0005)
	writeRec(0x0809, bofGlobals)

	// 2. CODEPAGE (1254 Turkish)
	cp := make([]byte, 2)
	binary.LittleEndian.PutUint16(cp, 1254)
	writeRec(0x0042, cp)

	// 3. SST (0x00FC) with UTF-16LE strings
	var sstBuf bytes.Buffer
	var sstHdr [8]byte
	binary.LittleEndian.PutUint32(sstHdr[0:4], totalRefs)
	binary.LittleEndian.PutUint32(sstHdr[4:8], uint32(len(sstStrings)))
	sstBuf.Write(sstHdr[:])
	for _, s := range sstStrings {
		u16 := utf16.Encode([]rune(s))
		var strHdr [3]byte
		binary.LittleEndian.PutUint16(strHdr[0:2], uint16(len(u16)))
		strHdr[2] = 0x01 // fHighByte = 1 (UTF-16LE)
		sstBuf.Write(strHdr[:])
		for _, ch := range u16 {
			var b2 [2]byte
			binary.LittleEndian.PutUint16(b2[:], ch)
			sstBuf.Write(b2[:])
		}
	}
	writeRec(0x00FC, sstBuf.Bytes())

	// 4. Workbook Globals EOF (0x000A)
	writeRec(0x000A, nil)

	// 5. Worksheet BOF (0x0809, version=0x0600, type=0x0010)
	bofSheet := make([]byte, 16)
	binary.LittleEndian.PutUint16(bofSheet[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bofSheet[2:4], 0x0010)
	writeRec(0x0809, bofSheet)

	// 6. LABELSST (0x00FD) records for each non-empty cell
	for rIdx, r := range rows {
		for cIdx, cell := range r {
			if cell == "" {
				continue
			}
			var lsst [10]byte
			binary.LittleEndian.PutUint16(lsst[0:2], uint16(rIdx))
			binary.LittleEndian.PutUint16(lsst[2:4], uint16(cIdx))
			binary.LittleEndian.PutUint16(lsst[4:6], 0)
			binary.LittleEndian.PutUint32(lsst[6:10], sstIndex[cell])
			writeRec(0x00FD, lsst[:])
		}
	}

	// 7. Worksheet EOF (0x000A)
	writeRec(0x000A, nil)

	// Pad Workbook stream to 4096 bytes (8 sectors of 512 bytes) so it uses standard FAT sectors
	streamBytes := wb.Bytes()
	if len(streamBytes) < 4096 {
		pad := make([]byte, 4096-len(streamBytes))
		streamBytes = append(streamBytes, pad...)
	} else if rem := len(streamBytes) % 512; rem != 0 {
		pad := make([]byte, 512-rem)
		streamBytes = append(streamBytes, pad...)
	}
	numStreamSectors := len(streamBytes) / 512

	// Construct OLE2 Compound File Binary container
	var cfb bytes.Buffer

	// Header (512 bytes)
	hdr := make([]byte, 512)
	copy(hdr[0:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
	binary.LittleEndian.PutUint16(hdr[24:26], 0x003E)
	binary.LittleEndian.PutUint16(hdr[26:28], 0x0003)
	binary.LittleEndian.PutUint16(hdr[28:30], 0xFFFE)
	binary.LittleEndian.PutUint16(hdr[30:32], 9) // 512-byte sectors
	binary.LittleEndian.PutUint16(hdr[32:34], 6) // 64-byte mini sectors
	binary.LittleEndian.PutUint32(hdr[44:48], 1) // 1 FAT sector
	binary.LittleEndian.PutUint32(hdr[48:52], 1) // Directory starts at sector 1
	binary.LittleEndian.PutUint32(hdr[56:60], 4096)
	binary.LittleEndian.PutUint32(hdr[60:64], 0xFFFFFFFE)
	binary.LittleEndian.PutUint32(hdr[68:72], 0xFFFFFFFE)
	binary.LittleEndian.PutUint32(hdr[76:80], 0) // DIFAT[0] = sector 0
	for pos := 80; pos < 512; pos += 4 {
		binary.LittleEndian.PutUint32(hdr[pos:pos+4], 0xFFFFFFFF)
	}
	cfb.Write(hdr)

	// Sector 0: FAT
	fat := make([]byte, 512)
	for pos := 0; pos < 512; pos += 4 {
		binary.LittleEndian.PutUint32(fat[pos:pos+4], 0xFFFFFFFF)
	}
	binary.LittleEndian.PutUint32(fat[0:4], 0xFFFFFFFD) // Sector 0 = FATSECT
	binary.LittleEndian.PutUint32(fat[4:8], 0xFFFFFFFE) // Sector 1 = Dir ENDOFCHAIN
	for s := 0; s < numStreamSectors; s++ {
		secID := 2 + s
		offset := secID * 4
		if s == numStreamSectors-1 {
			binary.LittleEndian.PutUint32(fat[offset:offset+4], 0xFFFFFFFE)
		} else {
			binary.LittleEndian.PutUint32(fat[offset:offset+4], uint32(secID+1))
		}
	}
	cfb.Write(fat)

	// Sector 1: Directory (4 entries x 128 bytes)
	dir := make([]byte, 512)
	writeDirName := func(entry []byte, name string) {
		u16 := utf16.Encode([]rune(name))
		for i, ch := range u16 {
			binary.LittleEndian.PutUint16(entry[i*2:i*2+2], ch)
		}
		binary.LittleEndian.PutUint16(entry[64:66], uint16((len(u16)+1)*2))
	}
	// Entry 0: Root Entry
	root := dir[0:128]
	writeDirName(root, "Root Entry")
	root[66] = 5 // Root Storage
	root[67] = 1 // Black
	binary.LittleEndian.PutUint32(root[68:72], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(root[72:76], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(root[76:80], 1) // Child = Entry 1
	binary.LittleEndian.PutUint32(root[116:120], 0xFFFFFFFE)

	// Entry 1: Workbook
	wbEntry := dir[128:256]
	writeDirName(wbEntry, "Workbook")
	wbEntry[66] = 2 // Stream
	wbEntry[67] = 1 // Black
	binary.LittleEndian.PutUint32(wbEntry[68:72], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(wbEntry[72:76], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(wbEntry[76:80], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(wbEntry[116:120], 2) // Start sector = 2
	binary.LittleEndian.PutUint32(wbEntry[120:124], uint32(len(streamBytes)))

	for e := 2; e < 4; e++ {
		unused := dir[e*128 : (e+1)*128]
		binary.LittleEndian.PutUint32(unused[68:72], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(unused[72:76], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(unused[76:80], 0xFFFFFFFF)
	}
	cfb.Write(dir)

	// Sectors 2+: Workbook stream
	cfb.Write(streamBytes)
	return cfb.Bytes()
}

func TestGSBOkulTaahhutListesiImportXLSAndDuplicates(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test_gsb_xls.db")
	db, err := database.Open(tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	svc := service.NewTournamentService(db)
	cat, err := svc.CreateCategory(1, models.SportVolleyball, "Yıldız Kızlar Voleybol", "groups")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Official GSB "Okul Taahhüt Listesi" structure with rows 1-6 metadata and row 7 headers
	gsbRows := [][]string{
		{"T.C. GENÇLİK VE SPOR BAKANLIĞI"},
		{"SPOR HİZMETLERİ GENEL MÜDÜRLÜĞÜ - OKUL SPORLARI"},
		{"OKUL TAAHHÜT LİSTESİ"},
		{"Rapor Tarihi: 26.10.2026"},
		{},
		{},
		{"Okul Adı", "İl", "İlçe", "Sezon", "Spor Dalı", "Kategori Türü", "Cinsiyet", "Eklenme Zamanı"},
		{"  Çukurova Sanayi Ortaokulu(A)  ", "Mersin", "Tarsus", "2026-2027", "Voleybol", "Yıldızlar", "Kız", "20.10.2026"},
		{"Atatürk Ortaokulu(A)", "Mersin", "Gülnar", "2026-2027", "Voleybol", "Yıldızlar", "Kız", "21.10.2026"},
		{"Atatürk Ortaokulu(A)", "Mersin", "Tarsus", "2026-2027", "Voleybol", "Yıldızlar", "Kız", "21.10.2026"},
		{"Cumhuriyet Ortaokulu(A)", "Mersin", "Akdeniz", "2026-2027", "Voleybol", "Yıldızlar", "Kız", "22.10.2026"},
		{"", "", "", "", "", "", "", ""}, // Empty row in the middle
		{"Cumhuriyet Ortaokulu(A)", "Mersin", "Akdeniz", "2026-2027", "Voleybol", "Yıldızlar", "Kız", "22.10.2026"}, // Duplicate row
	}

	xlsBytes := buildTestBIFF8XLS(gsbRows)

	summary, err := svc.ImportTeamsWithSummary(cat.ID, "Okul_Taahhut_Listesi.xls", bytes.NewReader(xlsBytes))
	if err != nil {
		t.Fatalf("ImportTeamsWithSummary(.xls) failed: %v", err)
	}

	if summary.ImportedCount != 4 {
		t.Fatalf("expected 4 imported teams, got %d (skipped=%d)", summary.ImportedCount, summary.SkippedCount)
	}
	if summary.SkippedCount != 2 {
		t.Fatalf("expected 2 skipped rows (1 duplicate + 1 empty), got %d", summary.SkippedCount)
	}

	expectedNames := []string{
		"Çukurova Sanayi Ortaokulu(A)",
		"Atatürk Ortaokulu(A) (Gülnar)",
		"Atatürk Ortaokulu(A) (Tarsus)",
		"Cumhuriyet Ortaokulu(A)",
	}
	for i, want := range expectedNames {
		if summary.Teams[i].DisplayName != want {
			t.Errorf("team[%d]: expected %q, got %q", i, want, summary.Teams[i].DisplayName)
		}
	}

	// Re-importing the exact same .xls file into the same category should skip all existing schools
	summary2, err := svc.ImportTeamsWithSummary(cat.ID, "Okul_Taahhut_Listesi.xls", bytes.NewReader(xlsBytes))
	if err != nil {
		t.Fatalf("second ImportTeamsWithSummary(.xls) failed: %v", err)
	}
	if summary2.ImportedCount != 0 {
		t.Fatalf("expected 0 imported teams on duplicate file upload, got %d", summary2.ImportedCount)
	}
	if summary2.SkippedCount != 6 {
		t.Fatalf("expected all 6 rows beneath header to be skipped on duplicate upload, got %d", summary2.SkippedCount)
	}
}

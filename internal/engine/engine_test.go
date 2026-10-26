package engine_test

import (
	"fmt"
	"testing"

	"tourna-desk/internal/engine"
	"tourna-desk/internal/engine/scoring"
	"tourna-desk/internal/models"
)

func TestGenerateRoundRobin_EvenAndOddTeams(t *testing.T) {
	teams4 := []models.Team{
		{ID: 1, SchoolName: "Atatürk Lisesi", TeamTag: "A"},
		{ID: 2, SchoolName: "Cumhuriyet Lisesi"},
		{ID: 3, SchoolName: "Fen Lisesi", TeamTag: "B"},
		{ID: 4, SchoolName: "Anadolu Lisesi"},
	}

	single4 := engine.GenerateRoundRobin(10, 20, teams4, false)
	if len(single4) != 6 {
		t.Fatalf("expected 6 matches for 4 teams single round-robin, got %d", len(single4))
	}
	for _, m := range single4 {
		if m.Status != models.StatusScheduled || m.HomeScore != 0 || m.AwayScore != 0 {
			t.Fatalf("expected unplayed match to be scheduled with 0-0 score, got %+v", m)
		}
	}

	double4 := engine.GenerateRoundRobin(10, 20, teams4, true)
	if len(double4) != 12 {
		t.Fatalf("expected 12 matches for 4 teams double round-robin, got %d", len(double4))
	}

	teams3 := teams4[:3]
	single3 := engine.GenerateRoundRobin(10, 20, teams3, false)
	if len(single3) != 6 {
		t.Fatalf("expected 6 match slots (including BYEs) for 3 teams, got %d", len(single3))
	}

	byeCount := 0
	for _, m := range single3 {
		if m.IsBye {
			byeCount++
			if m.AwayTeamName != "BAY" {
				t.Errorf("expected AwayTeamName to be BAY on bye match, got %s", m.AwayTeamName)
			}
		}
	}
	if byeCount != 3 {
		t.Fatalf("expected 3 BYE slots across 3 rounds, got %d", byeCount)
	}
}

func TestScoringStrategies_FinalScoresOnly(t *testing.T) {
	teams := []models.Team{
		{ID: 1, SchoolName: "Alpha School", TeamTag: "A"},
		{ID: 2, SchoolName: "Beta School"},
		{ID: 3, SchoolName: "Gamma School", TeamTag: "B"},
	}

	id1, id2, id3 := int64(1), int64(2), int64(3)

	// 1. Football Strategy (Win=3, Draw=1, Loss=0)
	footMatches := []models.Match{
		{HomeTeamID: &id1, AwayTeamID: &id2, HomeScore: 2, AwayScore: 1, Status: models.StatusCompleted},
		{HomeTeamID: &id2, AwayTeamID: &id3, HomeScore: 3, AwayScore: 0, Status: models.StatusCompleted},
		{HomeTeamID: &id1, AwayTeamID: &id3, HomeScore: 1, AwayScore: 1, Status: models.StatusCompleted},
	}
	footStrat := scoring.GetStrategy(models.SportFootball)
	footSt := footStrat.CalculateStandings(1, "Grup A", teams, footMatches)
	if footSt[0].TeamID != 1 || footSt[0].Points != 4 || footSt[0].TeamName != "Alpha School" {
		t.Fatalf("expected Alpha School first with 4 pts, got %+v", footSt[0])
	}
	if footSt[1].TeamID != 2 || footSt[1].Points != 3 {
		t.Fatalf("expected Beta second with 3 pts, got %+v", footSt[1])
	}

	// 2. Basketball Strategy (Win=2, Loss=1)
	baskMatches := []models.Match{
		{HomeTeamID: &id1, AwayTeamID: &id2, HomeScore: 70, AwayScore: 65, Status: models.StatusCompleted},
		{HomeTeamID: &id2, AwayTeamID: &id3, HomeScore: 80, AwayScore: 60, Status: models.StatusCompleted},
	}
	baskStrat := scoring.GetStrategy(models.SportBasketball)
	baskSt := baskStrat.CalculateStandings(1, "Grup A", teams, baskMatches)
	// Beta: 1L (1pt) + 1W (2pts) = 3 pts; Alpha: 1W (2pts); Gamma: 1L (1pt)
	if baskSt[0].TeamID != 2 || baskSt[0].Points != 3 {
		t.Fatalf("expected Beta first with 3 pts in basketball, got %+v", baskSt[0])
	}
	if baskSt[1].TeamID != 1 || baskSt[1].Points != 2 {
		t.Fatalf("expected Alpha second with 2 pts in basketball, got %+v", baskSt[1])
	}
	if baskSt[2].TeamID != 3 || baskSt[2].Points != 1 {
		t.Fatalf("expected Gamma third with 1 pt in basketball, got %+v", baskSt[2])
	}

	// 3. Volleyball Strategy (3-0/3-1 = 3pts, 3-2 = 2pts, 2-3 = 1pt, 0-3/1-3 = 0pts)
	volMatches := []models.Match{
		{
			HomeTeamID: &id1,
			AwayTeamID: &id2,
			HomeScore:  3,
			AwayScore:  2,
			Status:     models.StatusCompleted,
		},
		{
			HomeTeamID: &id3,
			AwayTeamID: &id2,
			HomeScore:  3,
			AwayScore:  0,
			Status:     models.StatusCompleted,
		},
	}
	volStrat := scoring.GetStrategy(models.SportVolleyball)
	volSt := volStrat.CalculateStandings(1, "Grup A", teams, volMatches)
	if volSt[0].TeamID != 3 || volSt[0].Points != 3 {
		t.Fatalf("expected Gamma #1 (1W, 3pts), got %+v", volSt[0])
	}
	if volSt[1].TeamID != 1 || volSt[1].Points != 2 {
		t.Fatalf("expected Alpha #2 (1W, 2pts), got %+v", volSt[1])
	}
	if volSt[2].TeamID != 2 || volSt[2].Points != 1 {
		t.Fatalf("expected Beta #3 (0W, 1pt), got %+v", volSt[2])
	}

	// 4. Handball Strategy (Win=2, Draw=1, Loss=0)
	handMatches := []models.Match{
		{HomeTeamID: &id1, AwayTeamID: &id2, HomeScore: 25, AwayScore: 22, Status: models.StatusCompleted},
		{HomeTeamID: &id2, AwayTeamID: &id3, HomeScore: 20, AwayScore: 20, Status: models.StatusCompleted},
		{HomeTeamID: &id1, AwayTeamID: &id3, HomeScore: 24, AwayScore: 24, Status: models.StatusCompleted},
	}
	handStrat := scoring.GetStrategy(models.SportHandball)
	handSt := handStrat.CalculateStandings(1, "Grup A", teams, handMatches)
	if handSt[0].TeamID != 1 || handSt[0].Points != 3 {
		t.Fatalf("expected Alpha #1 with 3 pts in handball, got %+v", handSt[0])
	}
}

func TestGenerateInitialGroups_SeededIsolation(t *testing.T) {
	// 12 teams, 3 seeded teams -> 3 groups of 4, each seeded team in a distinct group
	all12 := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	seeds := []int{2, 7, 11}

	for iter := 0; iter < 25; iter++ {
		groups, err := engine.GenerateInitialGroups(1, seeds, all12)
		if err != nil {
			t.Fatalf("GenerateInitialGroups failed: %v", err)
		}
		if len(groups) != 3 {
			t.Fatalf("expected 3 groups for 12 teams, got %d", len(groups))
		}

		seedSet := map[int]bool{2: true, 7: true, 11: true}
		seenTeams := make(map[int]bool)

		for _, g := range groups {
			if len(g.TeamIDs) != 4 {
				t.Fatalf("expected 4 teams in group %s, got %d", g.GroupName, len(g.TeamIDs))
			}
			seedsInGroup := 0
			for _, tid := range g.TeamIDs {
				if seenTeams[tid] {
					t.Fatalf("duplicate team %d across groups", tid)
				}
				seenTeams[tid] = true
				if seedSet[tid] {
					seedsInGroup++
				}
			}
			if seedsInGroup != 1 {
				t.Fatalf("expected exactly 1 seeded team in %s, got %d", g.GroupName, seedsInGroup)
			}
		}
	}

	// 16 teams, 3 seeded teams -> 4 groups of 4, seeded teams in 3 distinct groups
	all16 := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	groups16, err := engine.GenerateInitialGroups(1, seeds, all16)
	if err != nil {
		t.Fatalf("GenerateInitialGroups(16) failed: %v", err)
	}
	if len(groups16) != 4 {
		t.Fatalf("expected 4 groups for 16 teams, got %d", len(groups16))
	}
	seedSet := map[int]bool{2: true, 7: true, 11: true}
	for _, g := range groups16 {
		if len(g.TeamIDs) != 4 {
			t.Fatalf("expected 4 teams in group %s, got %d", g.GroupName, len(g.TeamIDs))
		}
		seedsInGroup := 0
		for _, tid := range g.TeamIDs {
			if seedSet[tid] {
				seedsInGroup++
			}
		}
		if seedsInGroup > 1 {
			t.Fatalf("seeded teams collided in %s (count=%d)", g.GroupName, seedsInGroup)
		}
	}
}

func TestPreparePotsAndExecuteNextRoundDraw_AvoidsSameGroup(t *testing.T) {
	// 4 completed stages (101, 102, 103, 104) -> 8 teams qualify (4 in Pot 1, 4 in Pot 2)
	// And 6 completed stages (201..206) -> 12 teams qualify (6 in Pot 1, 6 in Pot 2)
	standings4 := map[int][]models.Standing{
		101: {{Rank: 1, StageID: 101, StageName: "Grup A", TeamID: 1, TeamName: "A1"}, {Rank: 2, StageID: 101, StageName: "Grup A", TeamID: 2, TeamName: "A2"}},
		102: {{Rank: 1, StageID: 102, StageName: "Grup B", TeamID: 3, TeamName: "B1"}, {Rank: 2, StageID: 102, StageName: "Grup B", TeamID: 4, TeamName: "B2"}},
		103: {{Rank: 1, StageID: 103, StageName: "Grup C", TeamID: 5, TeamName: "C1"}, {Rank: 2, StageID: 103, StageName: "Grup C", TeamID: 6, TeamName: "C2"}},
		104: {{Rank: 1, StageID: 104, StageName: "Grup D", TeamID: 7, TeamName: "D1"}, {Rank: 2, StageID: 104, StageName: "Grup D", TeamID: 8, TeamName: "D2"}},
	}

	pots, err := engine.PrepareNextRoundPots([]int{101, 102, 103, 104}, standings4)
	if err != nil {
		t.Fatalf("PrepareNextRoundPots failed: %v", err)
	}
	if len(pots.Pot1) != 4 || len(pots.Pot2) != 4 {
		t.Fatalf("expected 4 teams in Pot1 and 4 in Pot2, got %d and %d", len(pots.Pot1), len(pots.Pot2))
	}

	pot1Set := map[int]bool{1: true, 3: true, 5: true, 7: true}
	pot2Set := map[int]bool{2: true, 4: true, 6: true, 8: true}

	for iter := 0; iter < 50; iter++ {
		drawn, err := engine.ExecuteNextRoundDraw(1, pots.Pot1, pots.Pot2, pots.PreviousGroupHistory)
		if err != nil {
			t.Fatalf("ExecuteNextRoundDraw failed on iter %d: %v", iter, err)
		}
		if len(drawn) != 2 {
			t.Fatalf("expected 2 groups of 4 teams, got %d groups", len(drawn))
		}
		for _, g := range drawn {
			if len(g.TeamIDs) != 4 {
				t.Fatalf("expected 4 teams in group %s, got %d", g.GroupName, len(g.TeamIDs))
			}
			p1Count, p2Count := 0, 0
			prevGroupsInThisGroup := make(map[int]bool)
			for _, tid := range g.TeamIDs {
				if pot1Set[tid] {
					p1Count++
				}
				if pot2Set[tid] {
					p2Count++
				}
				prevStg := pots.PreviousGroupHistory[tid]
				if prevGroupsInThisGroup[prevStg] {
					t.Fatalf("same-group constraint violated in %s: two teams from previous stage %d", g.GroupName, prevStg)
				}
				prevGroupsInThisGroup[prevStg] = true
			}
			if p1Count != 2 || p2Count != 2 {
				t.Fatalf("expected 2 teams from Pot1 and 2 from Pot2 in %s, got p1=%d p2=%d", g.GroupName, p1Count, p2Count)
			}
		}
	}

	// Test 6 groups (12 teams -> 3 groups of 4) where greedy without backtracking often hits dead-ends
	pot1_6 := []int{11, 21, 31, 41, 51, 61}
	pot2_6 := []int{12, 22, 32, 42, 52, 62}
	hist6 := map[int]int{
		11: 1, 12: 1,
		21: 2, 22: 2,
		31: 3, 32: 3,
		41: 4, 42: 4,
		51: 5, 52: 5,
		61: 6, 62: 6,
	}
	for iter := 0; iter < 50; iter++ {
		drawn6, err := engine.ExecuteNextRoundDraw(1, pot1_6, pot2_6, hist6)
		if err != nil {
			t.Fatalf("ExecuteNextRoundDraw(12 teams) failed on iter %d: %v", iter, err)
		}
		if len(drawn6) != 3 {
			t.Fatalf("expected 3 groups for 12 teams, got %d", len(drawn6))
		}
		for _, g := range drawn6 {
			seenStage := make(map[int]bool)
			for _, tid := range g.TeamIDs {
				st := hist6[tid]
				if seenStage[st] {
					t.Fatalf("clash in 12-team draw: stage %d appeared twice in %s", st, g.GroupName)
				}
				seenStage[st] = true
			}
		}
	}
}

func TestBuildFinalFour_CrossMatching(t *testing.T) {
	standings := map[int][]models.Standing{
		10: {
			{Rank: 1, StageID: 10, StageName: "Grup A", TeamID: 101, TeamName: "A-Winner"},
			{Rank: 2, StageID: 10, StageName: "Grup A", TeamID: 102, TeamName: "A-RunnerUp"},
		},
		20: {
			{Rank: 1, StageID: 20, StageName: "Grup B", TeamID: 201, TeamName: "B-Winner"},
			{Rank: 2, StageID: 20, StageName: "Grup B", TeamID: 202, TeamName: "B-RunnerUp"},
		},
	}

	ff, err := engine.BuildFinalFour(10, 20, standings)
	if err != nil {
		t.Fatalf("BuildFinalFour failed: %v", err)
	}
	if len(ff.Matches) != 4 {
		t.Fatalf("expected 4 matches in Final Four (SF1, SF2, 3rd, Final), got %d", len(ff.Matches))
	}

	sf1 := ff.Matches[0]
	if *sf1.HomeTeamID != 101 || *sf1.AwayTeamID != 202 {
		t.Fatalf("expected SF1 to be A1 (101) vs B2 (202), got %v vs %v", *sf1.HomeTeamID, *sf1.AwayTeamID)
	}

	sf2 := ff.Matches[1]
	if *sf2.HomeTeamID != 201 || *sf2.AwayTeamID != 102 {
		t.Fatalf("expected SF2 to be B1 (201) vs A2 (102), got %v vs %v", *sf2.HomeTeamID, *sf2.AwayTeamID)
	}

	third := ff.Matches[2]
	if third.BracketSlot != string(models.BracketSlotThirdPlace) {
		t.Fatalf("expected match 3 to be third_place, got %s", third.BracketSlot)
	}

	finalMatch := ff.Matches[3]
	if finalMatch.BracketSlot != string(models.BracketSlotFinal) {
		t.Fatalf("expected match 4 to be final, got %s", finalMatch.BracketSlot)
	}
}

func TestOfficialFixtureKeyRotation_4TeamAnd3Team(t *testing.T) {
	// Put the seeded team at index 2 initially to verify OrderGroupTeamsBySlot moves it strictly to Position 1
	teams4 := []models.Team{
		{ID: 10, SchoolName: "Zeta Lisesi"},
		{ID: 20, SchoolName: "Beta Lisesi"},
		{ID: 30, SchoolName: "Alfa Lisesi", IsSeeded: true},
		{ID: 40, SchoolName: "Delta Lisesi"},
	}

	matches4 := engine.GenerateRoundRobin(1, 1, teams4, false)
	if len(matches4) != 6 {
		t.Fatalf("expected 6 matches for 4-team group, got %d", len(matches4))
	}

	// Expected slot order: Team1=30 (Seeded), Team2=10, Team3=20, Team4=40
	// Round 1: 1-4 (30 vs 40), 2-3 (10 vs 20)
	// Round 2: 4-2 (40 vs 10), 3-1 (20 vs 30)
	// Round 3: 1-2 (30 vs 10), 3-4 (20 vs 40)
	expected4 := []struct {
		round int
		home  int64
		away  int64
	}{
		{1, 30, 40},
		{1, 10, 20},
		{2, 40, 10},
		{2, 20, 30},
		{3, 30, 10},
		{3, 20, 40},
	}

	for i, exp := range expected4 {
		m := matches4[i]
		if m.Round != exp.round || *m.HomeTeamID != exp.home || *m.AwayTeamID != exp.away {
			t.Fatalf("match %d mismatch: expected R%d (%d vs %d), got R%d (%d vs %d)",
				i+1, exp.round, exp.home, exp.away, m.Round, *m.HomeTeamID, *m.AwayTeamID)
		}
	}

	// 3-Team Group Key:
	// Round 1: 1-2 (3 BAY)
	// Round 2: 3-1 (2 BAY)
	// Round 3: 2-3 (1 BAY)
	teams3 := []models.Team{
		{ID: 1, SchoolName: "T1", SlotIndex: 1},
		{ID: 2, SchoolName: "T2", SlotIndex: 2},
		{ID: 3, SchoolName: "T3", SlotIndex: 3},
	}
	matches3 := engine.GenerateRoundRobin(1, 1, teams3, false)
	if len(matches3) != 6 {
		t.Fatalf("expected 6 match entries (3 played + 3 BYE) for 3-team group, got %d", len(matches3))
	}
	// Check played matches (indices 0, 2, 4) and BYE matches (indices 1, 3, 5)
	if *matches3[0].HomeTeamID != 1 || *matches3[0].AwayTeamID != 2 {
		t.Fatalf("expected R1 played match 1 vs 2, got %v vs %v", *matches3[0].HomeTeamID, *matches3[0].AwayTeamID)
	}
	if !matches3[1].IsBye || *matches3[1].HomeTeamID != 3 {
		t.Fatalf("expected R1 BYE for Team 3, got %+v", matches3[1])
	}
	if *matches3[2].HomeTeamID != 3 || *matches3[2].AwayTeamID != 1 {
		t.Fatalf("expected R2 played match 3 vs 1, got %v vs %v", *matches3[2].HomeTeamID, *matches3[2].AwayTeamID)
	}
	if !matches3[3].IsBye || *matches3[3].HomeTeamID != 2 {
		t.Fatalf("expected R2 BYE for Team 2, got %+v", matches3[3])
	}
	if *matches3[4].HomeTeamID != 2 || *matches3[4].AwayTeamID != 3 {
		t.Fatalf("expected R3 played match 2 vs 3, got %v vs %v", *matches3[4].HomeTeamID, *matches3[4].AwayTeamID)
	}
	if !matches3[5].IsBye || *matches3[5].HomeTeamID != 1 {
		t.Fatalf("expected R3 BYE for Team 1, got %+v", matches3[5])
	}
}

func TestSportSpecificForfeitRules(t *testing.T) {
	teams := []models.Team{
		{ID: 1, SchoolName: "Winner Team"},
		{ID: 2, SchoolName: "Forfeiting Team"},
		{ID: 3, SchoolName: "Regular Loser Team"},
	}
	id1, id2, id3 := int64(1), int64(2), int64(3)

	// 1. Basketball: Forfeit is 20-0, Winner gets 2 pts, Forfeit Loser gets 0 pts (Regular Loser gets 1 pt)
	baskMatches := []models.Match{
		{
			HomeTeamID:          &id1,
			AwayTeamID:          &id2,
			IsForfeit:           true,
			ForfeitWinnerTeamID: &id1,
			Status:              models.StatusCompleted,
		},
		{
			HomeTeamID: &id1,
			AwayTeamID: &id3,
			HomeScore:  60,
			AwayScore:  50,
			IsForfeit:  false,
			Status:     models.StatusCompleted,
		},
	}
	baskSt := scoring.GetStrategy(models.SportBasketball).CalculateStandings(1, "Grup A", teams, baskMatches)
	byTeam := make(map[int64]models.Standing)
	for _, s := range baskSt {
		byTeam[s.TeamID] = s
	}
	if byTeam[1].Points != 4 || byTeam[1].ScoreFor != 80 || byTeam[1].ScoreAgainst != 50 {
		t.Fatalf("expected basketball winner to have 4 pts and 80-50 score, got %+v", byTeam[1])
	}
	if byTeam[3].Points != 1 {
		t.Fatalf("expected basketball regular loser to receive 1 pt, got %+v", byTeam[3])
	}
	if byTeam[2].Points != 0 || byTeam[2].ScoreFor != 0 || byTeam[2].ScoreAgainst != 20 {
		t.Fatalf("expected basketball forfeiting team to receive 0 pts and 0-20 score, got %+v", byTeam[2])
	}

	// 2. Futsal: Forfeit is 5-0, Winner gets 3 pts, Loser gets 0 pts
	futsalMatch := []models.Match{
		{
			HomeTeamID:          &id1,
			AwayTeamID:          &id2,
			IsForfeit:           true,
			ForfeitWinnerTeamID: &id1,
			Status:              models.StatusCompleted,
		},
	}
	futsalSt := scoring.GetStrategy(models.SportFutsal).CalculateStandings(1, "Grup A", teams[:2], futsalMatch)
	if futsalSt[0].Points != 3 || futsalSt[0].ScoreFor != 5 || futsalSt[0].ScoreAgainst != 0 {
		t.Fatalf("expected futsal forfeit to be 5-0 (3 pts), got %+v", futsalSt[0])
	}

	// 3. Handball: Forfeit is 10-0, Winner gets 2 pts, Loser gets 0 pts
	handballSt := scoring.GetStrategy(models.SportHandball).CalculateStandings(1, "Grup A", teams[:2], futsalMatch)
	if handballSt[0].Points != 2 || handballSt[0].ScoreFor != 10 || handballSt[0].ScoreAgainst != 0 {
		t.Fatalf("expected handball forfeit to be 10-0 (2 pts), got %+v", handballSt[0])
	}
}

func TestFutsalPenaltyShootoutTiebreaker(t *testing.T) {
	teams := []models.Team{
		{ID: 1, SchoolName: "Anadolu Lisesi"},
		{ID: 2, SchoolName: "Fen Lisesi"},
	}
	id1, id2 := int64(1), int64(2)
	hp, ap := 3, 5 // Fen Lisesi wins penalty shootout 5-3 after 2-2 draw

	matches := []models.Match{
		{
			HomeTeamID:    &id1,
			AwayTeamID:    &id2,
			HomeScore:     2,
			AwayScore:     2,
			HomePenalties: &hp,
			AwayPenalties: &ap,
			Status:        models.StatusCompleted,
		},
	}

	st := scoring.GetStrategy(models.SportFutsal).CalculateStandings(1, "Grup A", teams, matches)
	if st[0].Points != 1 || st[1].Points != 1 {
		t.Fatalf("expected both teams to have 1 regulation point from draw, got %d and %d", st[0].Points, st[1].Points)
	}
	if st[0].ScoreFor != 2 || st[0].ScoreAgainst != 2 {
		t.Fatalf("expected regulation score 2-2 preserved in standings, got %d-%d", st[0].ScoreFor, st[0].ScoreAgainst)
	}
	if st[0].TeamID != 2 {
		t.Fatalf("expected Fen Lisesi (Team 2) to rank #1 via penalty shootout tiebreaker, got Team %d", st[0].TeamID)
	}
}

func TestGenerateKnockoutBracket_DynamicPowerOfTwoAndByes(t *testing.T) {
	cases := []struct {
		n               int
		expectedBracket int
		expectedByes    int
		expectedRounds  int
		expectedTotal   int
	}{
		{4, 4, 0, 2, 4},     // 2 SF + 3rd + Final
		{7, 8, 1, 3, 8},     // S=8, B=1 -> 4 QF (1 BYE + 3 playable) + 2 SF + 3rd + Final
		{8, 8, 0, 3, 8},     // 4 QF + 2 SF + 3rd + Final
		{10, 16, 6, 4, 16},  // S=16, B=6 -> 8 R16 (6 BYE + 2 playable) + 4 QF + 2 SF + 3rd + Final
		{12, 16, 4, 4, 16},  // S=16, B=4 -> 8 R16 (4 BYE + 4 playable) + 4 QF + 2 SF + 3rd + Final
		{16, 16, 0, 4, 16},  // 8 R16 + 4 QF + 2 SF + 3rd + Final
		{32, 32, 0, 5, 32},  // 16 R32 + 8 R16 + 4 QF + 2 SF + 3rd + Final
	}

	for _, tc := range cases {
		s, b := engine.CalculateBracketSize(tc.n)
		if s != tc.expectedBracket || b != tc.expectedByes {
			t.Fatalf("CalculateBracketSize(%d) = (%d, %d); expected (%d, %d)",
				tc.n, s, b, tc.expectedBracket, tc.expectedByes)
		}

		teams := make([]models.Team, tc.n)
		for i := 0; i < tc.n; i++ {
			teams[i] = models.Team{
				ID:         int64(i + 1),
				SchoolName: fmt.Sprintf("Okul %d", i+1),
			}
		}
		kb, err := engine.GenerateKnockoutBracket(1, 10, teams, []int64{1, 2}, true)
		if err != nil {
			t.Fatalf("GenerateKnockoutBracket(%d) failed: %v", tc.n, err)
		}
		if kb.BracketSize != tc.expectedBracket || kb.ByeCount != tc.expectedByes {
			t.Fatalf("for N=%d: got bracket (%d, %d), expected (%d, %d)",
				tc.n, kb.BracketSize, kb.ByeCount, tc.expectedBracket, tc.expectedByes)
		}
		if kb.NumRounds != tc.expectedRounds {
			t.Fatalf("for size %d: expected %d rounds, got %d", tc.n, tc.expectedRounds, kb.NumRounds)
		}
		if len(kb.Matches) != tc.expectedTotal {
			t.Fatalf("for size %d: expected %d total matches, got %d", tc.n, tc.expectedTotal, len(kb.Matches))
		}

		// Count Round 1 BYE matches and verify they have StatusByeAdvanced and auto-advance into Round 2
		r1Byes := 0
		r2AdvancedSlots := 0
		for _, m := range kb.Matches {
			if m.Round == 1 && m.IsBye {
				r1Byes++
				if m.Status != models.StatusByeAdvanced {
					t.Fatalf("expected BYE match status %s, got %s", models.StatusByeAdvanced, m.Status)
				}
				if m.WinnerTeamID == nil {
					t.Fatalf("expected WinnerTeamID to be set on BYE match")
				}
			}
			if m.Round == 2 && m.BracketSlot != string(models.BracketSlotThirdPlace) {
				if m.HomeTeamID != nil {
					r2AdvancedSlots++
				}
				if m.AwayTeamID != nil {
					r2AdvancedSlots++
				}
			}
		}
		if r1Byes != tc.expectedByes {
			t.Fatalf("for N=%d: expected %d BYE matches in Round 1, got %d", tc.n, tc.expectedByes, r1Byes)
		}
		if r2AdvancedSlots != tc.expectedByes {
			t.Fatalf("for N=%d: expected %d teams auto-advanced into Round 2 slots, got %d", tc.n, tc.expectedByes, r2AdvancedSlots)
		}
	}

	// Less than 2 teams should return error
	_, err := engine.GenerateKnockoutBracket(1, 10, make([]models.Team, 1), nil, false)
	if err == nil {
		t.Fatalf("expected error for 1 team, got nil")
	}
}

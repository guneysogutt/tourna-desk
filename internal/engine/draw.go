package engine

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"tourna-desk/internal/models"
)

// GroupDrawResult represents a single generated group from an initial or progression draw.
type GroupDrawResult struct {
	CategoryID int    `json:"category_id"`
	GroupIndex int    `json:"group_index"`
	GroupName  string `json:"group_name"`
	TeamIDs    []int  `json:"team_ids"`
	SeededIDs  []int  `json:"seeded_ids,omitempty"`
}

// PotAllocationResult holds the auto-allocated Pot 1 (group winners) and Pot 2 (runners-up)
// along with the mapping of each team to its previous group (stage ID).
type PotAllocationResult struct {
	Pot1                 []int                  `json:"pot1"`
	Pot2                 []int                  `json:"pot2"`
	Entries              []models.PotAssignment `json:"entries"`
	PreviousGroupHistory map[int]int            `json:"previous_group_history"`
}

// FinalFourBracket holds the instantiated matches and qualified teams for the Final Four stage.
type FinalFourBracket struct {
	GroupAStageID  int            `json:"group_a_stage_id"`
	GroupBStageID  int            `json:"group_b_stage_id"`
	QualifiedTeams []models.Team  `json:"qualified_teams"`
	Matches        []models.Match `json:"matches"`
}

// StageStandingsLookup is an optional hook allowing PrepareNextRoundPots and BuildFinalFour
// to resolve live stage standings when called with stage IDs only.
type StageStandingsLookup func(stageID int) ([]models.Standing, error)

var (
	standingsLookupMu sync.RWMutex
	standingsLookupFn StageStandingsLookup
)

// RegisterStageStandingsLookup registers a callback for resolving stage standings by stage ID.
func RegisterStageStandingsLookup(fn StageStandingsLookup) {
	standingsLookupMu.Lock()
	defer standingsLookupMu.Unlock()
	standingsLookupFn = fn
}

func resolveStandings(stageIDs []int, provided ...map[int][]models.Standing) (map[int][]models.Standing, error) {
	if len(provided) > 0 && provided[0] != nil {
		return provided[0], nil
	}
	standingsLookupMu.RLock()
	fn := standingsLookupFn
	standingsLookupMu.RUnlock()
	if fn == nil {
		return nil, errors.New("stage standings map not provided and no lookup registered")
	}
	res := make(map[int][]models.Standing, len(stageIDs))
	for _, id := range stageIDs {
		st, err := fn(id)
		if err != nil {
			return nil, err
		}
		res[id] = st
	}
	return res, nil
}

// GenerateInitialGroups creates 4-team groups (Grup A, Grup B, ...) for Stage 1,
// ensuring the seeded teams (Seri Başı) are placed into distinct groups first,
// and distributing the remaining teams via automated random draw.
func GenerateInitialGroups(categoryID int, seededTeamIDs []int, allTeamIDs []int) ([]GroupDrawResult, error) {
	if len(allTeamIDs) == 0 {
		return nil, errors.New("no teams available for initial draw")
	}

	// Deduplicate allTeamIDs while preserving order
	allSet := make(map[int]bool, len(allTeamIDs))
	uniqueAll := make([]int, 0, len(allTeamIDs))
	for _, id := range allTeamIDs {
		if id <= 0 || allSet[id] {
			continue
		}
		allSet[id] = true
		uniqueAll = append(uniqueAll, id)
	}
	if len(uniqueAll) < 2 {
		return nil, errors.New("at least 2 teams are required to generate groups")
	}

	// Deduplicate and validate seededTeamIDs
	seedSet := make(map[int]bool, len(seededTeamIDs))
	uniqueSeeds := make([]int, 0, len(seededTeamIDs))
	for _, id := range seededTeamIDs {
		if id <= 0 || seedSet[id] {
			continue
		}
		if !allSet[id] {
			return nil, fmt.Errorf("seeded team %d does not belong to this category", id)
		}
		seedSet[id] = true
		uniqueSeeds = append(uniqueSeeds, id)
	}

	// Calculate number of 4-team groups
	numGroups := (len(uniqueAll) + 3) / 4
	if numGroups < 1 {
		numGroups = 1
	}
	// Ensure there are enough groups so every seeded team is isolated in a distinct group
	if len(uniqueSeeds) > numGroups {
		numGroups = len(uniqueSeeds)
	}

	unseeded := make([]int, 0, len(uniqueAll)-len(uniqueSeeds))
	for _, id := range uniqueAll {
		if !seedSet[id] {
			unseeded = append(unseeded, id)
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(unseeded), func(i, j int) {
		unseeded[i], unseeded[j] = unseeded[j], unseeded[i]
	})

	groups := make([]GroupDrawResult, numGroups)
	for g := 0; g < numGroups; g++ {
		groups[g] = GroupDrawResult{
			CategoryID: categoryID,
			GroupIndex: g,
			GroupName:  FormatGroupName(g),
			TeamIDs:    []int{},
			SeededIDs:  []int{},
		}
	}

	// Step 1: Place each seeded team into a distinct group first
	for i, seedID := range uniqueSeeds {
		groups[i].TeamIDs = append(groups[i].TeamIDs, seedID)
		groups[i].SeededIDs = append(groups[i].SeededIDs, seedID)
	}

	// Calculate target group sizes (4 per group when divisible by 4, balanced otherwise)
	baseSize := len(uniqueAll) / numGroups
	rem := len(uniqueAll) % numGroups
	targetSize := make([]int, numGroups)
	for g := 0; g < numGroups; g++ {
		targetSize[g] = baseSize
		if g < rem {
			targetSize[g]++
		}
	}

	// Step 2: First bring any unseeded groups up to 1 team so all groups have 1 team before round-robin dealing
	unseededIdx := 0
	for g := len(uniqueSeeds); g < numGroups && unseededIdx < len(unseeded); g++ {
		if len(groups[g].TeamIDs) < targetSize[g] {
			groups[g].TeamIDs = append(groups[g].TeamIDs, unseeded[unseededIdx])
			unseededIdx++
		}
	}

	// Step 3: Distribute remaining teams across groups up to each group's target size (4 teams)
	for unseededIdx < len(unseeded) {
		progressed := false
		for g := 0; g < numGroups && unseededIdx < len(unseeded); g++ {
			if len(groups[g].TeamIDs) < targetSize[g] {
				groups[g].TeamIDs = append(groups[g].TeamIDs, unseeded[unseededIdx])
				unseededIdx++
				progressed = true
			}
		}
		if !progressed {
			break
		}
	}

	return groups, nil
}

// PrepareNextRoundPots automatically populates Pot 1 (group winners / 1st places)
// and Pot 2 (group runners-up / 2nd places) from the completed stages.
func PrepareNextRoundPots(completedStageIDs []int, standingsByStage ...map[int][]models.Standing) (*PotAllocationResult, error) {
	if len(completedStageIDs) == 0 {
		return nil, errors.New("no completed stages provided for pot allocation")
	}

	stMap, err := resolveStandings(completedStageIDs, standingsByStage...)
	if err != nil {
		return nil, err
	}

	res := &PotAllocationResult{
		Pot1:                 []int{},
		Pot2:                 []int{},
		Entries:              []models.PotAssignment{},
		PreviousGroupHistory: make(map[int]int),
	}

	seenTeams := make(map[int]bool)

	for _, stageID := range completedStageIDs {
		standings := stMap[stageID]
		if len(standings) == 0 {
			continue
		}

		// Rank 1 -> Pot 1 (1. Torba)
		first := standings[0]
		firstTeamID := int(first.TeamID)
		if !seenTeams[firstTeamID] {
			seenTeams[firstTeamID] = true
			res.Pot1 = append(res.Pot1, firstTeamID)
			res.PreviousGroupHistory[firstTeamID] = stageID
			stgID64 := int64(stageID)
			res.Entries = append(res.Entries, models.PotAssignment{
				StageID:    &stgID64,
				StageName:  first.StageName,
				TeamID:     first.TeamID,
				TeamName:   first.TeamName,
				SchoolName: first.SchoolName,
				TeamTag:    first.TeamTag,
				PotNumber:  1,
			})
		}

		// Rank 2 -> Pot 2 (2. Torba)
		if len(standings) >= 2 {
			second := standings[1]
			secondTeamID := int(second.TeamID)
			if !seenTeams[secondTeamID] {
				seenTeams[secondTeamID] = true
				res.Pot2 = append(res.Pot2, secondTeamID)
				res.PreviousGroupHistory[secondTeamID] = stageID
				stgID64 := int64(stageID)
				res.Entries = append(res.Entries, models.PotAssignment{
					StageID:    &stgID64,
					StageName:  second.StageName,
					TeamID:     second.TeamID,
					TeamName:   second.TeamName,
					SchoolName: second.SchoolName,
					TeamTag:    second.TeamTag,
					PotNumber:  2,
				})
			}
		}
	}

	return res, nil
}

// ExecuteNextRoundDraw generates the next round's 4-team groups from Pot 1 and Pot 2.
// Constraints enforced:
// 1. Pot Distribution: Each 4-team group receives 2 teams from Pot 1 and 2 teams from Pot 2 (or balanced quotas).
// 2. Same-Group Avoidance: Teams that advanced from the exact same group in the previous round
//    (previousGroupHistory[teamA] == previousGroupHistory[teamB] != 0) CANNOT be placed in the same group.
// 3. Backtracking / Re-try: Uses backtracking search with random shuffling to guarantee a valid draw.
func ExecuteNextRoundDraw(categoryID int, pot1 []int, pot2 []int, previousGroupHistory map[int]int) ([]GroupDrawResult, error) {
	totalTeams := len(pot1) + len(pot2)
	if totalTeams < 2 {
		return nil, errors.New("at least 2 teams across Pot 1 and Pot 2 are required to execute the draw")
	}

	if previousGroupHistory == nil {
		previousGroupHistory = make(map[int]int)
	}

	// Calculate number of target 4-team groups
	numGroups := (totalTeams + 3) / 4
	if numGroups < 1 {
		numGroups = 1
	}

	// Count how many teams in the pots share each previous stage
	stageFreq := make(map[int]int)
	for _, tid := range pot1 {
		if prev := previousGroupHistory[tid]; prev > 0 {
			stageFreq[prev]++
		}
	}
	for _, tid := range pot2 {
		if prev := previousGroupHistory[tid]; prev > 0 {
			stageFreq[prev]++
		}
	}

	// Maximum teams from the same previous stage allowed in a single target group:
	// Strictly 1 whenever stageFreq[prev] <= numGroups (standard Same-Group Avoidance constraint).
	maxPerGroupForStage := func(prevStage int) int {
		if prevStage <= 0 {
			return totalTeams
		}
		freq := stageFreq[prevStage]
		if freq <= numGroups {
			return 1
		}
		return (freq + numGroups - 1) / numGroups
	}

	// Compute balanced total quotas and per-pot quotas per group
	totalQuota := make([]int, numGroups)
	baseTotal := totalTeams / numGroups
	remTotal := totalTeams % numGroups
	for g := 0; g < numGroups; g++ {
		totalQuota[g] = baseTotal
		if g < remTotal {
			totalQuota[g]++
		}
	}

	pot1Quota := make([]int, numGroups)
	basePot1 := len(pot1) / numGroups
	remPot1 := len(pot1) % numGroups
	for g := 0; g < numGroups; g++ {
		pot1Quota[g] = basePot1
		if g < remPot1 {
			pot1Quota[g]++
		}
		if pot1Quota[g] > totalQuota[g] {
			pot1Quota[g] = totalQuota[g]
		}
	}

	pot2Quota := make([]int, numGroups)
	for g := 0; g < numGroups; g++ {
		pot2Quota[g] = totalQuota[g] - pot1Quota[g]
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Try randomized shuffles combined with complete backtracking solver
	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		p1 := make([]int, len(pot1))
		copy(p1, pot1)
		p2 := make([]int, len(pot2))
		copy(p2, pot2)

		rng.Shuffle(len(p1), func(i, j int) { p1[i], p1[j] = p1[j], p1[i] })
		rng.Shuffle(len(p2), func(i, j int) { p2[i], p2[j] = p2[j], p2[i] })

		// Build combined candidate list: first all Pot 1 teams, then all Pot 2 teams
		type potTeam struct {
			teamID int
			pot    int // 1 or 2
		}
		items := make([]potTeam, 0, totalTeams)
		for _, id := range p1 {
			items = append(items, potTeam{teamID: id, pot: 1})
		}
		for _, id := range p2 {
			items = append(items, potTeam{teamID: id, pot: 2})
		}

		groupTeams := make([][]int, numGroups)
		groupP1Count := make([]int, numGroups)
		groupP2Count := make([]int, numGroups)
		for g := 0; g < numGroups; g++ {
			groupTeams[g] = make([]int, 0, totalQuota[g])
		}

		canPlace := func(groupIdx int, pt potTeam) bool {
			if pt.pot == 1 && groupP1Count[groupIdx] >= pot1Quota[groupIdx] {
				return false
			}
			if pt.pot == 2 && groupP2Count[groupIdx] >= pot2Quota[groupIdx] {
				return false
			}
			prevStage := previousGroupHistory[pt.teamID]
			if prevStage > 0 {
				limit := maxPerGroupForStage(prevStage)
				sameCount := 0
				for _, existingTeamID := range groupTeams[groupIdx] {
					if previousGroupHistory[existingTeamID] == prevStage {
						sameCount++
						if sameCount >= limit {
							return false
						}
					}
				}
			}
			return true
		}

		var backtrack func(idx int) bool
		backtrack = func(idx int) bool {
			if idx == len(items) {
				return true
			}
			pt := items[idx]

			// Randomize group trial order at each step for fair draw distribution
			groupOrder := rng.Perm(numGroups)
			for _, g := range groupOrder {
				if canPlace(g, pt) {
					groupTeams[g] = append(groupTeams[g], pt.teamID)
					if pt.pot == 1 {
						groupP1Count[g]++
					} else {
						groupP2Count[g]++
					}

					if backtrack(idx + 1) {
						return true
					}

					// Backtrack
					groupTeams[g] = groupTeams[g][:len(groupTeams[g])-1]
					if pt.pot == 1 {
						groupP1Count[g]--
					} else {
						groupP2Count[g]--
					}
				}
			}
			return false
		}

		if backtrack(0) {
			results := make([]GroupDrawResult, numGroups)
			for g := 0; g < numGroups; g++ {
				results[g] = GroupDrawResult{
					CategoryID: categoryID,
					GroupIndex: g,
					GroupName:  FormatGroupName(g),
					TeamIDs:    groupTeams[g],
				}
			}
			return results, nil
		}
	}

	return nil, errors.New("could not find a valid group assignment satisfying all pot and same-group avoidance constraints")
}

// BuildFinalFour instantiates the Semi-Final, 3rd Place, and Final matches using cross-matching rules:
// - Semi-Final 1: Grup A (1.) vs Grup B (2.)
// - Semi-Final 2: Grup B (1.) vs Grup A (2.)
// - 3rd Place Match: Loser of Semi-Final 1 vs Loser of Semi-Final 2
// - Final (Championship Match): Winner of Semi-Final 1 vs Winner of Semi-Final 2
func BuildFinalFour(groupAStageID, groupBStageID int, standingsByStage ...map[int][]models.Standing) (*FinalFourBracket, error) {
	if groupAStageID <= 0 || groupBStageID <= 0 || groupAStageID == groupBStageID {
		return nil, errors.New("two distinct source group stage IDs (Group A and Group B) are required for Final Four")
	}

	stMap, err := resolveStandings([]int{groupAStageID, groupBStageID}, standingsByStage...)
	if err != nil {
		return nil, err
	}

	standingsA := stMap[groupAStageID]
	standingsB := stMap[groupBStageID]
	if len(standingsA) < 2 {
		return nil, fmt.Errorf("Group A (stage %d) must have at least 2 ranked teams to build Final Four", groupAStageID)
	}
	if len(standingsB) < 2 {
		return nil, fmt.Errorf("Group B (stage %d) must have at least 2 ranked teams to build Final Four", groupBStageID)
	}

	a1 := standingsA[0]
	a2 := standingsA[1]
	b1 := standingsB[0]
	b2 := standingsB[1]

	stgA64 := int64(groupAStageID)
	stgB64 := int64(groupBStageID)

	a1ID := a1.TeamID
	a2ID := a2.TeamID
	b1ID := b1.TeamID
	b2ID := b2.TeamID

	qualifiedTeams := []models.Team{
		{ID: a1.TeamID, SchoolName: a1.SchoolName, TeamTag: a1.TeamTag, DisplayName: a1.TeamName},
		{ID: b2.TeamID, SchoolName: b2.SchoolName, TeamTag: b2.TeamTag, DisplayName: b2.TeamName},
		{ID: b1.TeamID, SchoolName: b1.SchoolName, TeamTag: b1.TeamTag, DisplayName: b1.TeamName},
		{ID: a2.TeamID, SchoolName: a2.SchoolName, TeamTag: a2.TeamTag, DisplayName: a2.TeamName},
	}

	matches := []models.Match{
		{
			Round:             1,
			MatchNumber:       1,
			BracketSlot:       string(models.BracketSlotSemiFinal1),
			SourceHomeStageID: &stgA64,
			SourceHomeRank:    1,
			SourceAwayStageID: &stgB64,
			SourceAwayRank:    2,
			HomeTeamID:        &a1ID,
			AwayTeamID:        &b2ID,
			HomeTeamName:      a1.TeamName,
			AwayTeamName:      b2.TeamName,
			Status:            models.StatusScheduled,
			Notes:             fmt.Sprintf("Yarı Final 1 (%s 1. vs %s 2.)", nonEmptyStageName(a1.StageName, "Grup A"), nonEmptyStageName(b2.StageName, "Grup B")),
		},
		{
			Round:             1,
			MatchNumber:       2,
			BracketSlot:       string(models.BracketSlotSemiFinal2),
			SourceHomeStageID: &stgB64,
			SourceHomeRank:    1,
			SourceAwayStageID: &stgA64,
			SourceAwayRank:    2,
			HomeTeamID:        &b1ID,
			AwayTeamID:        &a2ID,
			HomeTeamName:      b1.TeamName,
			AwayTeamName:      a2.TeamName,
			Status:            models.StatusScheduled,
			Notes:             fmt.Sprintf("Yarı Final 2 (%s 1. vs %s 2.)", nonEmptyStageName(b1.StageName, "Grup B"), nonEmptyStageName(a2.StageName, "Grup A")),
		},
		{
			Round:        2,
			MatchNumber:  3,
			BracketSlot:  string(models.BracketSlotThirdPlace),
			HomeTeamID:   nil,
			AwayTeamID:   nil,
			HomeTeamName: "Yarı Final 1 Mağlubu",
			AwayTeamName: "Yarı Final 2 Mağlubu",
			Status:       models.StatusScheduled,
			Notes:        "3.'lük Maçı",
		},
		{
			Round:        2,
			MatchNumber:  4,
			BracketSlot:  string(models.BracketSlotFinal),
			HomeTeamID:   nil,
			AwayTeamID:   nil,
			HomeTeamName: "Yarı Final 1 Galibi",
			AwayTeamName: "Yarı Final 2 Galibi",
			Status:       models.StatusScheduled,
			Notes:        "Final (Şampiyonluk Maçı)",
		},
	}

	return &FinalFourBracket{
		GroupAStageID:  groupAStageID,
		GroupBStageID:  groupBStageID,
		QualifiedTeams: qualifiedTeams,
		Matches:        matches,
	}, nil
}

// FormatGroupName returns "Grup A", "Grup B", "Grup C", etc. for 0-based index.
func FormatGroupName(groupIndex int) string {
	if groupIndex >= 0 && groupIndex < 26 {
		return fmt.Sprintf("Grup %c", 'A'+groupIndex)
	}
	return fmt.Sprintf("Grup %d", groupIndex+1)
}

func nonEmptyStageName(name, fallback string) string {
	if name != "" {
		return name
	}
	return fallback
}

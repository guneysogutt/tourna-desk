package engine

import (
	"fmt"
	"math/rand"
	"time"

	"tourna-desk/internal/models"
)

// KnockoutBracket holds the instantiated single-elimination bracket tree with dynamic power-of-2 sizing and BYE slots.
type KnockoutBracket struct {
	BracketSize int            `json:"bracket_size"`
	ByeCount    int            `json:"bye_count"`
	NumRounds   int            `json:"num_rounds"`
	Teams       []models.Team  `json:"teams"`
	Matches     []models.Match `json:"matches"`
}

// CalculateBracketSize scales a single-elimination tournament to the nearest power-of-two bracket size
// S in {4, 8, 16, 32, 64, ...} such that S >= teamCount (minimum S = 4), and returns S and B = S - N.
func CalculateBracketSize(teamCount int) (bracketSize int, byeCount int) {
	if teamCount <= 0 {
		return 0, 0
	}
	bracketSize = 4
	for bracketSize < teamCount {
		bracketSize *= 2
	}
	byeCount = bracketSize - teamCount
	return bracketSize, byeCount
}

// buildSeededMatchOrder returns the 0-based Round 1 match indices (0..numMatches-1)
// in standard tournament seed priority order, alternating across top and bottom halves
// and quarters so that seeded teams and BYE slots are maximally separated.
// Example for numMatches=4 (S=8): [0, 3, 1, 2]
// Example for numMatches=8 (S=16): [0, 7, 3, 4, 1, 6, 2, 5]
func buildSeededMatchOrder(numMatches int) []int {
	if numMatches <= 1 {
		return []int{0}
	}
	seeds := []int{1, 2}
	for len(seeds) < numMatches {
		next := make([]int, 0, len(seeds)*2)
		sum := len(seeds)*2 + 1
		for _, s := range seeds {
			next = append(next, s, sum-s)
		}
		seeds = next
	}
	order := make([]int, len(seeds))
	for i, s := range seeds {
		order[i] = s - 1
	}
	return order
}

func knockoutSlotAndTitle(matchesInRound int, indexInRound int) (slot string, note string) {
	switch matchesInRound {
	case 32:
		return fmt.Sprintf("r64_%d", indexInRound+1), fmt.Sprintf("Son 64 - Maç %d", indexInRound+1)
	case 16:
		return fmt.Sprintf("r32_%d", indexInRound+1), fmt.Sprintf("Son 32 - Maç %d", indexInRound+1)
	case 8:
		return fmt.Sprintf("r16_%d", indexInRound+1), fmt.Sprintf("Son 16 - Maç %d", indexInRound+1)
	case 4:
		return fmt.Sprintf("qf_%d", indexInRound+1), fmt.Sprintf("Çeyrek Final %d", indexInRound+1)
	case 2:
		if indexInRound == 0 {
			return string(models.BracketSlotSemiFinal1), "Yarı Final 1"
		}
		return string(models.BracketSlotSemiFinal2), "Yarı Final 2"
	default:
		return fmt.Sprintf("r1_%d", indexInRound+1), fmt.Sprintf("1. Tur - Maç %d", indexInRound+1)
	}
}

// GenerateKnockoutBracket builds a dynamic single-elimination knockout bracket for any team count N >= 2.
// It rounds up to the nearest power of 2 (S >= 4), distributes B = S - N BYE (BAY) slots systematically
// across top and bottom bracket halves, marks BYE matches as StatusByeAdvanced, and automatically advances
// teams paired with a BYE into Round 2.
func GenerateKnockoutBracket(categoryID, stageID int64, teams []models.Team, seededTeamIDs []int64, randomize bool) (*KnockoutBracket, error) {
	n := len(teams)
	if n < 2 {
		return nil, fmt.Errorf("knockout bracket requires at least 2 teams; got %d", n)
	}

	bracketSize, byeCount := CalculateBracketSize(n)
	round1MatchCount := bracketSize / 2

	pool := make([]models.Team, n)
	for i, t := range teams {
		if t.DisplayName == "" {
			t.DisplayName = models.FormatTeamDisplayName(t.SchoolName)
		}
		pool[i] = t
	}

	seedSet := make(map[int64]bool, len(seededTeamIDs))
	for _, sid := range seededTeamIDs {
		seedSet[sid] = true
	}
	if len(seedSet) == 0 {
		for _, t := range pool {
			if t.IsSeeded {
				seedSet[t.ID] = true
			}
		}
	}

	var prioritized []models.Team
	if randomize {
		var seeds, unseeded []models.Team
		for _, t := range pool {
			if seedSet[t.ID] {
				seeds = append(seeds, t)
			} else {
				unseeded = append(unseeded, t)
			}
		}
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		rng.Shuffle(len(unseeded), func(i, j int) {
			unseeded[i], unseeded[j] = unseeded[j], unseeded[i]
		})
		prioritized = append(seeds, unseeded...)
	} else {
		var seeds, unseeded []models.Team
		for _, t := range pool {
			if seedSet[t.ID] {
				seeds = append(seeds, t)
			} else {
				unseeded = append(unseeded, t)
			}
		}
		prioritized = append(seeds, unseeded...)
	}

	// Place teams into Round 1 match slots using standard bracket seeding order
	matchOrder := buildSeededMatchOrder(round1MatchCount)
	homeSlots := make([]*models.Team, round1MatchCount)
	awaySlots := make([]*models.Team, round1MatchCount)

	teamIdx := 0
	// 1. Fill Home slots of Round 1 matches in priority order
	for _, mIdx := range matchOrder {
		if teamIdx < len(prioritized) {
			tm := prioritized[teamIdx]
			homeSlots[mIdx] = &tm
			teamIdx++
		}
	}

	// 2. The first `byeCount` matches in `matchOrder` receive a BYE in their Away slot.
	// The remaining matches receive the remaining teams in reverse priority order.
	for i := round1MatchCount - 1; i >= byeCount; i-- {
		mIdx := matchOrder[i]
		if teamIdx < len(prioritized) {
			tm := prioritized[teamIdx]
			awaySlots[mIdx] = &tm
			teamIdx++
		}
	}

	// Build ordered team list reflecting bracket slot order
	orderedTeams := make([]models.Team, 0, n)
	slotCounter := 1
	for i := 0; i < round1MatchCount; i++ {
		if homeSlots[i] != nil {
			tm := *homeSlots[i]
			tm.SlotIndex = slotCounter
			slotCounter++
			orderedTeams = append(orderedTeams, tm)
		}
		if awaySlots[i] != nil {
			tm := *awaySlots[i]
			tm.SlotIndex = slotCounter
			slotCounter++
			orderedTeams = append(orderedTeams, tm)
		}
	}

	// Calculate number of rounds: log2(bracketSize)
	totalRounds := 0
	for s := bracketSize; s > 1; s /= 2 {
		totalRounds++
	}

	// Build rounds as a 2D slice so we can propagate BYE advancements cleanly
	rounds := make([][]models.Match, totalRounds)
	var thirdPlaceMatch models.Match

	matchNumber := 1

	// Round 1
	rounds[0] = make([]models.Match, round1MatchCount)
	for i := 0; i < round1MatchCount; i++ {
		slot, note := knockoutSlotAndTitle(round1MatchCount, i)
		nextSlot := "home"
		if i%2 == 1 {
			nextSlot = "away"
		}

		home := homeSlots[i]
		away := awaySlots[i]

		var hID, aID, winID *int64
		homeName := "TBD"
		awayName := "BAY"
		isBye := false
		status := models.StatusScheduled

		if home != nil {
			idCopy := home.ID
			hID = &idCopy
			homeName = home.DisplayName
		}
		if away != nil {
			idCopy := away.ID
			aID = &idCopy
			awayName = away.DisplayName
		} else if home != nil {
			// Paired with BYE -> automatically advances
			isBye = true
			status = models.StatusByeAdvanced
			idCopy := home.ID
			winID = &idCopy
			awayName = "BAY"
		}

		rounds[0][i] = models.Match{
			CategoryID:    categoryID,
			StageID:       stageID,
			Round:         1,
			MatchNumber:   matchNumber,
			BracketSlot:   slot,
			HomeTeamID:    hID,
			AwayTeamID:    aID,
			HomeTeamName:  homeName,
			AwayTeamName:  awayName,
			IsBye:         isBye,
			WinnerTeamID:  winID,
			NextMatchSlot: nextSlot,
			Status:        status,
			Notes:         note,
		}
		matchNumber++
	}

	// Subsequent Rounds (r = 2 .. totalRounds)
	matchesInRound := round1MatchCount
	for r := 2; r <= totalRounds; r++ {
		matchesInRound /= 2
		prevRound := rounds[r-2]

		if r == totalRounds {
			// Final Round: 3rd Place Match + Final Match
			thirdPlaceMatch = models.Match{
				CategoryID:   categoryID,
				StageID:      stageID,
				Round:        r,
				MatchNumber:  matchNumber,
				BracketSlot:  string(models.BracketSlotThirdPlace),
				HomeTeamName: "Yarı Final 1 Mağlubu",
				AwayTeamName: "Yarı Final 2 Mağlubu",
				Status:       models.StatusScheduled,
				Notes:        "3.'lük Maçı",
			}
			matchNumber++

			finalMatch := models.Match{
				CategoryID:   categoryID,
				StageID:      stageID,
				Round:        r,
				MatchNumber:  matchNumber,
				BracketSlot:  string(models.BracketSlotFinal),
				HomeTeamName: "Yarı Final 1 Galibi",
				AwayTeamName: "Yarı Final 2 Galibi",
				Status:       models.StatusScheduled,
				Notes:        "Final (Şampiyonluk Maçı)",
			}
			matchNumber++

			// Propagate if Semi-Final 1 or 2 was a BYE (e.g., when N=2 or N=3)
			if len(prevRound) >= 2 {
				if prevRound[0].IsBye && prevRound[0].WinnerTeamID != nil {
					idCopy := *prevRound[0].WinnerTeamID
					finalMatch.HomeTeamID = &idCopy
					finalMatch.HomeTeamName = prevRound[0].HomeTeamName
					thirdPlaceMatch.HomeTeamName = "BAY"
				}
				if prevRound[1].IsBye && prevRound[1].WinnerTeamID != nil {
					idCopy := *prevRound[1].WinnerTeamID
					finalMatch.AwayTeamID = &idCopy
					finalMatch.AwayTeamName = prevRound[1].HomeTeamName
					thirdPlaceMatch.AwayTeamName = "BAY"
				}
			}

			rounds[r-1] = []models.Match{finalMatch}
			break
		}

		rounds[r-1] = make([]models.Match, matchesInRound)
		for i := 0; i < matchesInRound; i++ {
			slot, note := knockoutSlotAndTitle(matchesInRound, i)
			nextSlot := "home"
			if i%2 == 1 {
				nextSlot = "away"
			}

			childHome := prevRound[i*2]
			childAway := prevRound[i*2+1]

			var hID, aID *int64
			homePlaceholder := fmt.Sprintf("%s Galibi", childHome.Notes)
			awayPlaceholder := fmt.Sprintf("%s Galibi", childAway.Notes)

			// Automatically advance any child match that had a BYE
			if childHome.IsBye && childHome.WinnerTeamID != nil {
				idCopy := *childHome.WinnerTeamID
				hID = &idCopy
				homePlaceholder = childHome.HomeTeamName
			}
			if childAway.IsBye && childAway.WinnerTeamID != nil {
				idCopy := *childAway.WinnerTeamID
				aID = &idCopy
				awayPlaceholder = childAway.HomeTeamName
			}

			rounds[r-1][i] = models.Match{
				CategoryID:    categoryID,
				StageID:       stageID,
				Round:         r,
				MatchNumber:   matchNumber,
				BracketSlot:   slot,
				HomeTeamID:    hID,
				AwayTeamID:    aID,
				HomeTeamName:  homePlaceholder,
				AwayTeamName:  awayPlaceholder,
				NextMatchSlot: nextSlot,
				Status:        models.StatusScheduled,
				Notes:         note,
			}
			matchNumber++
		}
	}

	// Flatten matches in round order (with 3rd place match right before Final in the last round)
	var allMatches []models.Match
	for r := 0; r < totalRounds; r++ {
		if r == totalRounds-1 {
			allMatches = append(allMatches, thirdPlaceMatch)
		}
		allMatches = append(allMatches, rounds[r]...)
	}

	return &KnockoutBracket{
		BracketSize: bracketSize,
		ByeCount:    byeCount,
		NumRounds:   totalRounds,
		Teams:       orderedTeams,
		Matches:     allMatches,
	}, nil
}

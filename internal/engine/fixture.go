package engine

import (
	"sort"

	"tourna-desk/internal/models"
)

// fixtureKeyPair defines a deterministic 1-based slot pairing (HomeSlot vs AwaySlot) within a round.
type fixtureKeyPair struct {
	round    int
	homeSlot int // 1-based index (0 means BAY)
	awaySlot int // 1-based index (0 means BAY)
}

// fourTeamKey implements the official 4-team group fixture key (4'lü Grup Anahtarı):
// Round 1: 1-4, 2-3
// Round 2: 4-2, 3-1
// Round 3: 1-2, 3-4
var fourTeamKey = []fixtureKeyPair{
	{round: 1, homeSlot: 1, awaySlot: 4},
	{round: 1, homeSlot: 2, awaySlot: 3},
	{round: 2, homeSlot: 4, awaySlot: 2},
	{round: 2, homeSlot: 3, awaySlot: 1},
	{round: 3, homeSlot: 1, awaySlot: 2},
	{round: 3, homeSlot: 3, awaySlot: 4},
}

// threeTeamKey implements the official 3-team group fixture key (3'lü Grup / BAY Anahtarı):
// Round 1: 1-2 (Team 3 BAY)
// Round 2: 3-1 (Team 2 BAY)
// Round 3: 2-3 (Team 1 BAY)
var threeTeamKey = []fixtureKeyPair{
	{round: 1, homeSlot: 1, awaySlot: 2},
	{round: 1, homeSlot: 3, awaySlot: 0}, // Team 3 BAY
	{round: 2, homeSlot: 3, awaySlot: 1},
	{round: 2, homeSlot: 2, awaySlot: 0}, // Team 2 BAY
	{round: 3, homeSlot: 2, awaySlot: 3},
	{round: 3, homeSlot: 1, awaySlot: 0}, // Team 1 BAY
}

// OrderGroupTeamsBySlot orders a group's teams deterministically by SlotIndex (1..4),
// ensuring any seeded team (IsSeeded) is placed strictly at Position 1 (index 0)
// when explicit SlotIndex values are not yet assigned.
func OrderGroupTeamsBySlot(teams []models.Team) []models.Team {
	pool := make([]models.Team, len(teams))
	hasExplicitSlots := false
	for i, t := range teams {
		if t.DisplayName == "" {
			t.DisplayName = models.FormatTeamDisplayName(t.SchoolName, t.TeamTag)
		}
		if t.SlotIndex > 0 {
			hasExplicitSlots = true
		}
		pool[i] = t
	}

	if hasExplicitSlots {
		sort.SliceStable(pool, func(i, j int) bool {
			si := pool[i].SlotIndex
			sj := pool[j].SlotIndex
			if si <= 0 {
				si = 999
			}
			if sj <= 0 {
				sj = 999
			}
			return si < sj
		})
	} else {
		// Ensure seeded team (Seri Başı) is strictly in Position 1 (index 0)
		seedIdx := -1
		for i, t := range pool {
			if t.IsSeeded {
				seedIdx = i
				break
			}
		}
		if seedIdx > 0 {
			seededTeam := pool[seedIdx]
			copy(pool[1:seedIdx+1], pool[0:seedIdx])
			pool[0] = seededTeam
		}
	}

	for i := range pool {
		pool[i].SlotIndex = i + 1
	}
	return pool
}

// GenerateRoundRobin generates matches using official deterministic federation fixture keys
// for 4-team and 3-team groups, and falls back to Berger rotation for other group sizes.
// Supports both single and double round-robin formats.
func GenerateRoundRobin(categoryID, stageID int64, teams []models.Team, doubleRoundRobin bool) []models.Match {
	if len(teams) < 2 {
		return nil
	}

	pool := OrderGroupTeamsBySlot(teams)

	var firstHalf []models.Match
	numRounds := 0

	if len(pool) == 4 {
		numRounds = 3
		firstHalf = buildMatchesFromKey(categoryID, stageID, pool, fourTeamKey)
	} else if len(pool) == 3 {
		numRounds = 3
		firstHalf = buildMatchesFromKey(categoryID, stageID, pool, threeTeamKey)
	} else {
		firstHalf, numRounds = buildBergerFallback(categoryID, stageID, pool)
	}

	if !doubleRoundRobin {
		return firstHalf
	}

	allMatches := make([]models.Match, 0, len(firstHalf)*2)
	allMatches = append(allMatches, firstHalf...)
	matchNumber := len(firstHalf) + 1

	for _, m := range firstHalf {
		secondHalfMatch := models.Match{
			CategoryID:   categoryID,
			StageID:      stageID,
			Round:        m.Round + numRounds,
			MatchNumber:  matchNumber,
			HomeTeamID:   m.AwayTeamID,
			AwayTeamID:   m.HomeTeamID,
			HomeTeamName: m.AwayTeamName,
			AwayTeamName: m.HomeTeamName,
			IsBye:        m.IsBye,
			HomeScore:    0,
			AwayScore:    0,
			Status:       models.StatusScheduled,
		}
		if m.IsBye {
			secondHalfMatch.HomeTeamID = m.HomeTeamID
			secondHalfMatch.AwayTeamID = m.AwayTeamID
			secondHalfMatch.HomeTeamName = m.HomeTeamName
			secondHalfMatch.AwayTeamName = m.AwayTeamName
		}
		allMatches = append(allMatches, secondHalfMatch)
		matchNumber++
	}

	return allMatches
}

func buildMatchesFromKey(categoryID, stageID int64, pool []models.Team, key []fixtureKeyPair) []models.Match {
	matches := make([]models.Match, 0, len(key))
	for idx, pair := range key {
		matchNumber := idx + 1
		home := pool[pair.homeSlot-1]
		hID := home.ID
		homeIDPtr := &hID

		if pair.awaySlot == 0 {
			// BYE / BAY slot
			matches = append(matches, models.Match{
				CategoryID:   categoryID,
				StageID:      stageID,
				Round:        pair.round,
				MatchNumber:  matchNumber,
				HomeTeamID:   homeIDPtr,
				AwayTeamID:   nil,
				HomeTeamName: home.DisplayName,
				AwayTeamName: "BAY",
				IsBye:        true,
				HomeScore:    0,
				AwayScore:    0,
				Status:       models.StatusScheduled,
			})
			continue
		}

		away := pool[pair.awaySlot-1]
		aID := away.ID
		awayIDPtr := &aID

		matches = append(matches, models.Match{
			CategoryID:   categoryID,
			StageID:      stageID,
			Round:        pair.round,
			MatchNumber:  matchNumber,
			HomeTeamID:   homeIDPtr,
			AwayTeamID:   awayIDPtr,
			HomeTeamName: home.DisplayName,
			AwayTeamName: away.DisplayName,
			IsBye:        false,
			HomeScore:    0,
			AwayScore:    0,
			Status:       models.StatusScheduled,
		})
	}
	return matches
}

func buildBergerFallback(categoryID, stageID int64, pool []models.Team) ([]models.Match, int) {
	work := make([]models.Team, len(pool))
	copy(work, pool)

	if len(work)%2 != 0 {
		work = append(work, models.Team{
			ID:          0,
			SchoolName:  "BAY",
			DisplayName: "BAY",
		})
	}

	n := len(work)
	numRounds := n - 1
	half := n / 2

	var firstHalf []models.Match
	matchNumber := 1

	rotation := make([]models.Team, n)
	copy(rotation, work)

	for round := 1; round <= numRounds; round++ {
		for i := 0; i < half; i++ {
			t1 := rotation[i]
			t2 := rotation[n-1-i]

			home := t1
			away := t2
			if i == 0 && round%2 == 0 {
				home = t2
				away = t1
			} else if i > 0 && (round+i)%2 == 0 {
				home = t2
				away = t1
			}

			isBye := home.ID == 0 || away.ID == 0
			if home.ID == 0 && away.ID != 0 {
				home, away = away, home
			}

			var homeID, awayID *int64
			if home.ID != 0 {
				id := home.ID
				homeID = &id
			}
			if away.ID != 0 {
				id := away.ID
				awayID = &id
			}

			m := models.Match{
				CategoryID:   categoryID,
				StageID:      stageID,
				Round:        round,
				MatchNumber:  matchNumber,
				HomeTeamID:   homeID,
				AwayTeamID:   awayID,
				HomeTeamName: home.DisplayName,
				AwayTeamName: away.DisplayName,
				IsBye:        isBye,
				HomeScore:    0,
				AwayScore:    0,
				Status:       models.StatusScheduled,
			}
			firstHalf = append(firstHalf, m)
			matchNumber++
		}

		last := rotation[n-1]
		copy(rotation[2:], rotation[1:n-1])
		rotation[1] = last
	}

	return firstHalf, numRounds
}

package scoring

import (
	"math"
	"sort"
	"strings"

	"tourna-desk/internal/models"
)

// VolleyballStrategy implements Volleyball rules strictly based on final set scores (home_score & away_score):
// 3-0 / 3-1 Win = 3 pts; 3-2 Win = 2 pts; 2-3 Loss = 1 pt; 0-3 / 1-3 Loss = 0 pts.
// Ranking: Total Wins -> Points -> Set Ratio -> Set Diff.
type VolleyballStrategy struct{}

func (s *VolleyballStrategy) Sport() models.SportSlug {
	return models.SportVolleyball
}

func (s *VolleyballStrategy) CalculateStandings(stageID int64, stageName string, teams []models.Team, matches []models.Match) []models.Standing {
	stats := make(map[int64]*models.Standing, len(teams))

	for _, t := range teams {
		displayName := t.DisplayName
		if displayName == "" {
			displayName = models.FormatTeamDisplayName(t.SchoolName, t.TeamTag)
		}
		stats[t.ID] = &models.Standing{
			StageID:    stageID,
			StageName:  stageName,
			TeamID:     t.ID,
			TeamName:   displayName,
			SchoolName: t.SchoolName,
			TeamTag:    t.TeamTag,
			IsSeeded:   t.IsSeeded,
		}
	}

	for _, m := range matches {
		if m.IsBye || m.HomeTeamID == nil || m.AwayTeamID == nil {
			continue
		}
		if m.Status != models.StatusCompleted {
			continue
		}

		homeID := *m.HomeTeamID
		awayID := *m.AwayTeamID
		home, okH := stats[homeID]
		away, okA := stats[awayID]
		if !okH || !okA {
			continue
		}

		homeSets, awaySets, _, _ := ResolveMatchOutcome(s.Sport(), m)

		home.Played++
		away.Played++
		home.SetsWon += homeSets
		home.SetsLost += awaySets
		away.SetsWon += awaySets
		away.SetsLost += homeSets

		// Also populate ScoreFor / ScoreAgainst with sets won/lost for unified table/excel display
		home.ScoreFor += homeSets
		home.ScoreAgainst += awaySets
		away.ScoreFor += awaySets
		away.ScoreAgainst += homeSets

		if homeSets > awaySets {
			home.Won++
			away.Lost++
			if homeSets-awaySets >= 2 {
				// e.g. 3-0 or 3-1 (or 2-0)
				home.Points += 3
			} else {
				// e.g. 3-2 (or 2-1)
				home.Points += 2
				away.Points += 1
			}
		} else if awaySets > homeSets {
			away.Won++
			home.Lost++
			if awaySets-homeSets >= 2 {
				// e.g. 0-3 or 1-3 (or 0-2)
				away.Points += 3
			} else {
				// e.g. 2-3 (or 1-2)
				away.Points += 2
				home.Points += 1
			}
		}
	}

	result := make([]models.Standing, 0, len(teams))
	for _, t := range teams {
		st := stats[t.ID]
		st.ScoreDiff = st.SetsWon - st.SetsLost
		if st.SetsLost > 0 {
			st.SetRatio = math.Round((float64(st.SetsWon)/float64(st.SetsLost))*1000) / 1000
		} else if st.SetsWon > 0 {
			st.SetRatio = float64(st.SetsWon)
		}
		result = append(result, *st)
	}

	// Ranking: Total Wins -> Points -> Set Ratio -> Set Difference
	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		if a.Won != b.Won {
			return a.Won > b.Won
		}
		if a.Points != b.Points {
			return a.Points > b.Points
		}

		srCmp := compareRatio(a.SetsWon, a.SetsLost, b.SetsWon, b.SetsLost)
		if srCmp != 0 {
			return srCmp > 0
		}

		if a.ScoreDiff != b.ScoreDiff {
			return a.ScoreDiff > b.ScoreDiff
		}

		return strings.ToLower(a.TeamName) < strings.ToLower(b.TeamName)
	})

	for i := range result {
		result[i].Rank = i + 1
	}

	return result
}

func compareRatio(numA, denA, numB, denB int) int {
	if denA == 0 && denB == 0 {
		if numA != numB {
			if numA > numB {
				return 1
			}
			return -1
		}
		return 0
	}
	if denA == 0 {
		if numA > 0 {
			return 1
		}
		if numB > 0 {
			return -1
		}
		return 0
	}
	if denB == 0 {
		if numB > 0 {
			return -1
		}
		if numA > 0 {
			return 1
		}
		return 0
	}
	rA := float64(numA) / float64(denA)
	rB := float64(numB) / float64(denB)
	if math.Abs(rA-rB) > 1e-6 {
		if rA > rB {
			return 1
		}
		return -1
	}
	return 0
}

package scoring

import (
	"sort"
	"strings"

	"tourna-desk/internal/models"
)

// FootballStrategy implements Football/Futsal rules:
// Win = 3, Draw = 1, Loss = 0.
// Official Forfeit (Hükmen): Football = 3-0, Futsal = 5-0.
// Ranking: Points -> Goal Difference -> Goals For -> Head-to-Head.
type FootballStrategy struct {
	sportSlug models.SportSlug
}

func (s *FootballStrategy) Sport() models.SportSlug {
	if s.sportSlug == models.SportFutsal {
		return models.SportFutsal
	}
	return models.SportFootball
}

func (s *FootballStrategy) CalculateStandings(stageID int64, stageName string, teams []models.Team, matches []models.Match) []models.Standing {
	stats := make(map[int64]*models.Standing, len(teams))
	h2h := make(map[int64]map[int64]*h2hRecord, len(teams))
	penaltyWins := make(map[int64]int, len(teams))
	penaltyDiff := make(map[int64]int, len(teams))
	penaltyFor := make(map[int64]int, len(teams))

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
		h2h[t.ID] = make(map[int64]*h2hRecord)
	}

	getH2H := func(a, b int64) *h2hRecord {
		if _, ok := h2h[a]; !ok {
			h2h[a] = make(map[int64]*h2hRecord)
		}
		if rec, ok := h2h[a][b]; ok {
			return rec
		}
		rec := &h2hRecord{}
		h2h[a][b] = rec
		return rec
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

		homeScore, awayScore, _, _ := ResolveMatchOutcome(s.Sport(), m)

		home.Played++
		away.Played++
		home.ScoreFor += homeScore
		home.ScoreAgainst += awayScore
		away.ScoreFor += awayScore
		away.ScoreAgainst += homeScore

		recH := getH2H(homeID, awayID)
		recA := getH2H(awayID, homeID)
		recH.scoreFor += homeScore
		recH.scoreDiff += homeScore - awayScore
		recA.scoreFor += awayScore
		recA.scoreDiff += awayScore - homeScore

		if homeScore > awayScore {
			home.Won++
			home.Points += 3
			away.Lost++
			recH.points += 3
			recH.wins++
		} else if homeScore < awayScore {
			away.Won++
			away.Points += 3
			home.Lost++
			recA.points += 3
			recA.wins++
		} else {
			home.Drawn++
			away.Drawn++
			home.Points += 1
			away.Points += 1
			recH.points += 1
			recA.points += 1

			// Record penalty shootout outcome on drawn regulation matches (e.g., Futsal group draws)
			if m.HomePenalties != nil && m.AwayPenalties != nil {
				hp := *m.HomePenalties
				ap := *m.AwayPenalties
				recH.penaltyFor += hp
				recH.penaltyDiff += hp - ap
				recA.penaltyFor += ap
				recA.penaltyDiff += ap - hp
				penaltyFor[homeID] += hp
				penaltyFor[awayID] += ap
				penaltyDiff[homeID] += hp - ap
				penaltyDiff[awayID] += ap - hp
				if hp > ap {
					recH.penaltyWins++
					penaltyWins[homeID]++
				} else if ap > hp {
					recA.penaltyWins++
					penaltyWins[awayID]++
				}
			}
		}
	}

	result := make([]models.Standing, 0, len(teams))
	for _, t := range teams {
		st := stats[t.ID]
		st.ScoreDiff = st.ScoreFor - st.ScoreAgainst
		result = append(result, *st)
	}

	// Ranking: Points -> Head-to-Head (including penalty shootouts in Futsal) / Goal Difference -> Goals For
	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		if a.Points != b.Points {
			return a.Points > b.Points
		}

		recAB := getH2H(a.TeamID, b.TeamID)
		recBA := getH2H(b.TeamID, a.TeamID)

		// If the two teams played a drawn match resolved by penalties, factor H2H penalty result into tiebreaking
		if a.ScoreDiff != b.ScoreDiff {
			return a.ScoreDiff > b.ScoreDiff
		}
		if a.ScoreFor != b.ScoreFor {
			return a.ScoreFor > b.ScoreFor
		}

		if recAB.points != recBA.points {
			return recAB.points > recBA.points
		}
		if recAB.scoreDiff != recBA.scoreDiff {
			return recAB.scoreDiff > recBA.scoreDiff
		}
		if recAB.scoreFor != recBA.scoreFor {
			return recAB.scoreFor > recBA.scoreFor
		}
		if recAB.penaltyWins != recBA.penaltyWins {
			return recAB.penaltyWins > recBA.penaltyWins
		}
		if recAB.penaltyDiff != recBA.penaltyDiff {
			return recAB.penaltyDiff > recBA.penaltyDiff
		}
		if recAB.penaltyFor != recBA.penaltyFor {
			return recAB.penaltyFor > recBA.penaltyFor
		}

		if penaltyWins[a.TeamID] != penaltyWins[b.TeamID] {
			return penaltyWins[a.TeamID] > penaltyWins[b.TeamID]
		}
		if penaltyDiff[a.TeamID] != penaltyDiff[b.TeamID] {
			return penaltyDiff[a.TeamID] > penaltyDiff[b.TeamID]
		}
		if penaltyFor[a.TeamID] != penaltyFor[b.TeamID] {
			return penaltyFor[a.TeamID] > penaltyFor[b.TeamID]
		}

		return strings.ToLower(a.TeamName) < strings.ToLower(b.TeamName)
	})

	for i := range result {
		result[i].Rank = i + 1
	}

	return result
}

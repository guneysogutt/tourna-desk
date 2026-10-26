package scoring

import (
	"sort"
	"strings"

	"tourna-desk/internal/models"
)

// HandballStrategy implements Handball (Hentbol) rules:
// Win = 2, Draw = 1, Loss = 0.
// Ranking: Points -> Goal Difference -> Goals For -> Head-to-Head.
type HandballStrategy struct{}

func (s *HandballStrategy) Sport() models.SportSlug {
	return models.SportHandball
}

func (s *HandballStrategy) CalculateStandings(stageID int64, stageName string, teams []models.Team, matches []models.Match) []models.Standing {
	stats := make(map[int64]*models.Standing, len(teams))
	h2h := make(map[int64]map[int64]*h2hRecord, len(teams))

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
			home.Points += 2
			away.Lost++
			recH.points += 2
		} else if homeScore < awayScore {
			away.Won++
			away.Points += 2
			home.Lost++
			recA.points += 2
		} else {
			home.Drawn++
			away.Drawn++
			home.Points += 1
			away.Points += 1
			recH.points += 1
			recA.points += 1
		}
	}

	result := make([]models.Standing, 0, len(teams))
	for _, t := range teams {
		st := stats[t.ID]
		st.ScoreDiff = st.ScoreFor - st.ScoreAgainst
		result = append(result, *st)
	}

	// Ranking: Points -> Goal Difference -> Goals For -> Head-to-Head
	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.ScoreDiff != b.ScoreDiff {
			return a.ScoreDiff > b.ScoreDiff
		}
		if a.ScoreFor != b.ScoreFor {
			return a.ScoreFor > b.ScoreFor
		}

		recAB := getH2H(a.TeamID, b.TeamID)
		recBA := getH2H(b.TeamID, a.TeamID)
		if recAB.points != recBA.points {
			return recAB.points > recBA.points
		}
		if recAB.scoreDiff != recBA.scoreDiff {
			return recAB.scoreDiff > recBA.scoreDiff
		}
		if recAB.scoreFor != recBA.scoreFor {
			return recAB.scoreFor > recBA.scoreFor
		}

		return strings.ToLower(a.TeamName) < strings.ToLower(b.TeamName)
	})

	for i := range result {
		result[i].Rank = i + 1
	}

	return result
}

package scoring

import (
	"tourna-desk/internal/models"
)

// ScoringStrategy defines the pluggable interface for sport-specific match evaluation and standings ranking.
type ScoringStrategy interface {
	Sport() models.SportSlug
	CalculateStandings(stageID int64, stageName string, teams []models.Team, matches []models.Match) []models.Standing
}

// GetStrategy returns the concrete ScoringStrategy for the given sport slug.
func GetStrategy(sport models.SportSlug) ScoringStrategy {
	switch sport {
	case models.SportBasketball:
		return &BasketballStrategy{}
	case models.SportVolleyball:
		return &VolleyballStrategy{}
	case models.SportHandball:
		return &HandballStrategy{}
	case models.SportFutsal:
		return &FootballStrategy{sportSlug: models.SportFutsal}
	default:
		return &FootballStrategy{sportSlug: models.SportFootball}
	}
}

// OfficialForfeitScore returns the official federation forfeit (hükmen) score for each sport discipline:
// - Futsal:     5 - 0
// - Football:   3 - 0
// - Basketball: 20 - 0
// - Volleyball: 3 - 0
// - Handball:   10 - 0
func OfficialForfeitScore(sport models.SportSlug) (winnerScore int, loserScore int) {
	switch sport {
	case models.SportFutsal:
		return 5, 0
	case models.SportBasketball:
		return 20, 0
	case models.SportVolleyball:
		return 3, 0
	case models.SportHandball:
		return 10, 0
	case models.SportFootball:
		return 3, 0
	default:
		return 3, 0
	}
}

// ResolveMatchOutcome returns the effective homeScore, awayScore, and whether home or away forfeited.
func ResolveMatchOutcome(sport models.SportSlug, m models.Match) (homeScore int, awayScore int, homeForfeited bool, awayForfeited bool) {
	if !m.IsForfeit {
		return m.HomeScore, m.AwayScore, false, false
	}

	winScore, loseScore := OfficialForfeitScore(sport)

	if m.ForfeitWinnerTeamID != nil {
		if m.HomeTeamID != nil && *m.ForfeitWinnerTeamID == *m.HomeTeamID {
			return winScore, loseScore, false, true
		}
		if m.AwayTeamID != nil && *m.ForfeitWinnerTeamID == *m.AwayTeamID {
			return loseScore, winScore, true, false
		}
	}

	if m.AwayScore > m.HomeScore {
		return loseScore, winScore, true, false
	}
	return winScore, loseScore, false, true
}

type h2hRecord struct {
	points      int
	scoreDiff   int
	scoreFor    int
	wins        int
	penaltyWins int
	penaltyDiff int
	penaltyFor  int
}

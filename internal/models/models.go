package models

import (
	"strings"
	"time"
)

type SportSlug string

const (
	SportFootball   SportSlug = "football"
	SportFutsal     SportSlug = "futsal"
	SportBasketball SportSlug = "basketball"
	SportVolleyball SportSlug = "volleyball"
	SportHandball   SportSlug = "handball"
)

type MatchStatus string

const (
	StatusScheduled   MatchStatus = "scheduled"
	StatusCompleted   MatchStatus = "completed"
	StatusByeAdvanced MatchStatus = "bye_advanced"
)

type BracketSlot string

const (
	BracketSlotNone       BracketSlot = ""
	BracketSlotSemiFinal1 BracketSlot = "semi_final_1"
	BracketSlotSemiFinal2 BracketSlot = "semi_final_2"
	BracketSlotThirdPlace BracketSlot = "third_place"
	BracketSlotFinal      BracketSlot = "final"
)

type Tournament struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	StartDate   string     `json:"start_date"`
	EndDate     string     `json:"end_date"`
	CreatedAt   time.Time  `json:"created_at"`
	Categories  []Category `json:"categories,omitempty"`
}

type Sport struct {
	Slug   SportSlug `json:"slug"`
	NameTR string    `json:"name_tr"`
	NameEN string    `json:"name_en"`
}

type Venue struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	SupportedSports []string `json:"supported_sports"`
}

type Category struct {
	ID           int64     `json:"id"`
	TournamentID int64     `json:"tournament_id"`
	SportSlug    SportSlug `json:"sport_slug"`
	Name         string    `json:"name"`
	Format       string    `json:"format"` // "groups" or "knockout"
	Status       string    `json:"status"` // "AWAITING_DRAW", "ACTIVE"
	CreatedAt    time.Time `json:"created_at"`
	Stages       []Stage   `json:"stages,omitempty"`
	Teams        []Team    `json:"teams,omitempty"`
}

type Stage struct {
	ID               int64     `json:"id"`
	CategoryID       int64     `json:"category_id"`
	Name             string    `json:"name"`
	StageType        string    `json:"stage_type"` // "group", "final_group", "final_four", "knockout"
	Status           string    `json:"status"`     // "AWAITING_DRAW", "ACTIVE"
	RoundNumber      int       `json:"round_number"`
	ParentStageID    *int64    `json:"parent_stage_id,omitempty"`
	DoubleRoundRobin bool      `json:"double_round_robin"`
	CreatedAt        time.Time `json:"created_at"`
	Teams            []Team    `json:"teams,omitempty"`
	Matches          []Match   `json:"matches,omitempty"`
}

type Team struct {
	ID          int64     `json:"id"`
	CategoryID  int64     `json:"category_id"`
	StageID     *int64    `json:"stage_id"`
	SlotIndex   int       `json:"slot_index"`
	SchoolName  string    `json:"school_name"`
	TeamTag     string    `json:"team_tag"`
	DisplayName string    `json:"display_name"`
	IsSeeded    bool      `json:"is_seeded"`
	CreatedAt   time.Time `json:"created_at"`
}

// FormatTeamDisplayName returns the trimmed school_name as the canonical team display name.
func FormatTeamDisplayName(schoolName string, _ ...string) string {
	return strings.TrimSpace(schoolName)
}

type Match struct {
	ID                int64       `json:"id"`
	CategoryID        int64       `json:"category_id"`
	StageID           int64       `json:"stage_id"`
	StageName         string      `json:"stage_name,omitempty"`
	Round             int         `json:"round"`
	MatchNumber       int         `json:"match_number"`
	BracketSlot       string      `json:"bracket_slot,omitempty"`
	SourceHomeStageID *int64      `json:"source_home_stage_id,omitempty"`
	SourceHomeRank    int         `json:"source_home_rank,omitempty"`
	SourceAwayStageID *int64      `json:"source_away_stage_id,omitempty"`
	SourceAwayRank    int         `json:"source_away_rank,omitempty"`
	HomeTeamID        *int64      `json:"home_team_id"`
	AwayTeamID        *int64      `json:"away_team_id"`
	HomeTeamName      string      `json:"home_team_name"`
	AwayTeamName      string      `json:"away_team_name"`
	IsBye               bool        `json:"is_bye"`
	IsForfeit           bool        `json:"is_forfeit"`
	ForfeitWinnerTeamID *int64      `json:"forfeit_winner_team_id,omitempty"`
	HomeScore           int         `json:"home_score"`
	AwayScore           int         `json:"away_score"`
	HomePenalties       *int        `json:"home_penalties"`
	AwayPenalties       *int        `json:"away_penalties"`
	NextMatchID         *int64      `json:"next_match_id,omitempty"`
	NextMatchSlot       string      `json:"next_match_slot,omitempty"`
	WinnerTeamID        *int64      `json:"winner_team_id,omitempty"`
	Status              MatchStatus `json:"status"`
	Venue             string      `json:"venue"`
	Court             string      `json:"court"`
	ScheduledAt       string      `json:"scheduled_at"`
	Notes             string      `json:"notes"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type PotAssignment struct {
	ID          int64     `json:"id"`
	CategoryID  int64     `json:"category_id"`
	RoundNumber int       `json:"round_number"`
	StageID     *int64    `json:"stage_id"`
	StageName   string    `json:"stage_name,omitempty"`
	TeamID      int64     `json:"team_id"`
	TeamName    string    `json:"team_name"`
	SchoolName  string    `json:"school_name"`
	TeamTag     string    `json:"team_tag"`
	PotNumber   int       `json:"pot_number"` // 1 or 2
	CreatedAt   time.Time `json:"created_at"`
}

type Standing struct {
	Rank         int     `json:"rank"`
	StageID      int64   `json:"stage_id"`
	StageName    string  `json:"stage_name"`
	TeamID       int64   `json:"team_id"`
	TeamName     string  `json:"team_name"`
	SchoolName   string  `json:"school_name"`
	TeamTag      string  `json:"team_tag"`
	IsSeeded     bool    `json:"is_seeded"`
	Played       int     `json:"played"`
	Won          int     `json:"won"`
	Drawn        int     `json:"drawn"`
	Lost         int     `json:"lost"`
	Points       int     `json:"points"`
	ScoreFor     int     `json:"score_for"`
	ScoreAgainst int     `json:"score_against"`
	ScoreDiff    int     `json:"score_diff"`
	SetsWon      int     `json:"sets_won"`
	SetsLost     int     `json:"sets_lost"`
	SetRatio     float64 `json:"set_ratio"`
}

type CategoryDetail struct {
	Category       Category             `json:"category"`
	TournamentName string               `json:"tournament_name"`
	Stages         []Stage              `json:"stages"`
	Teams          []Team               `json:"teams"`
	Matches        []Match              `json:"matches"`
	Standings      map[int64][]Standing `json:"standings"`
	AllStandings   []Standing           `json:"all_standings"`
	PotAssignments []PotAssignment      `json:"pot_assignments"`
	CurrentRound   int                  `json:"current_round"`
}

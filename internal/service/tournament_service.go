package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tourna-desk/internal/database"
	"tourna-desk/internal/engine"
	"tourna-desk/internal/engine/scoring"
	"tourna-desk/internal/models"
)

type TournamentService struct {
	db *database.DB
}

func NewTournamentService(db *database.DB) *TournamentService {
	svc := &TournamentService{db: db}
	engine.RegisterStageStandingsLookup(svc.lookupStageStandings)
	return svc
}

func (s *TournamentService) lookupStageStandings(stageID int) ([]models.Standing, error) {
	var catID int64
	var stageName string
	var sportSlug string
	err := s.db.Conn.QueryRow(`
		SELECT st.category_id, st.name, c.sport_slug
		FROM stages st
		JOIN categories c ON c.id = st.category_id
		WHERE st.id = ?`, stageID).Scan(&catID, &stageName, &sportSlug)
	if err != nil {
		return nil, err
	}
	teams, err := s.listTeamsByStage(int64(stageID))
	if err != nil {
		return nil, err
	}
	matches, err := s.listMatchesByStage(int64(stageID))
	if err != nil {
		return nil, err
	}
	strategy := scoring.GetStrategy(models.SportSlug(sportSlug))
	return strategy.CalculateStandings(int64(stageID), stageName, teams, matches), nil
}

func (s *TournamentService) GetSupportedSports() []models.Sport {
	return []models.Sport{
		{Slug: models.SportFootball, NameTR: "Futbol", NameEN: "Football"},
		{Slug: models.SportFutsal, NameTR: "Futsal", NameEN: "Futsal"},
		{Slug: models.SportBasketball, NameTR: "Basketbol", NameEN: "Basketball"},
		{Slug: models.SportVolleyball, NameTR: "Voleybol", NameEN: "Volleyball"},
		{Slug: models.SportHandball, NameTR: "Hentbol", NameEN: "Handball"},
	}
}

func isValidSportSlug(slug models.SportSlug) bool {
	switch slug {
	case models.SportFootball, models.SportFutsal, models.SportBasketball, models.SportVolleyball, models.SportHandball:
		return true
	default:
		return false
	}
}

func (s *TournamentService) ListTournaments() ([]models.Tournament, error) {
	rows, err := s.db.Conn.Query(`
		SELECT id, name, description, location, start_date, end_date, created_at
		FROM tournaments
		ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tournaments []models.Tournament
	for rows.Next() {
		var t models.Tournament
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Location, &t.StartDate, &t.EndDate, &t.CreatedAt); err != nil {
			return nil, err
		}
		tournaments = append(tournaments, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range tournaments {
		cats, err := s.ListCategoriesByTournament(tournaments[i].ID)
		if err != nil {
			return nil, err
		}
		tournaments[i].Categories = cats
	}

	if tournaments == nil {
		tournaments = []models.Tournament{}
	}
	return tournaments, nil
}

func (s *TournamentService) CreateTournament(name, description, location, startDate, endDate string) (*models.Tournament, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("tournament name is required")
	}
	res, err := s.db.Conn.Exec(`
		INSERT INTO tournaments (name, description, location, start_date, end_date)
		VALUES (?, ?, ?, ?, ?)`,
		name, strings.TrimSpace(description), strings.TrimSpace(location), startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.Tournament{
		ID:          id,
		Name:        name,
		Description: description,
		Location:    location,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedAt:   time.Now(),
		Categories:  []models.Category{},
	}, nil
}

func (s *TournamentService) DeleteTournament(id int64) error {
	_, err := s.db.Conn.Exec(`DELETE FROM tournaments WHERE id = ?`, id)
	return err
}

func normalizeSportCode(raw string) string {
	v := strings.ToUpper(strings.TrimSpace(raw))
	switch v {
	case "BASKETBALL", "BASKETBOL":
		return "BASKETBOL"
	case "VOLLEYBALL", "VOLEYBOL":
		return "VOLEYBOL"
	case "FOOTBALL", "FUTBOL":
		return "FUTBOL"
	case "FUTSAL":
		return "FUTSAL"
	case "HANDBALL", "HENTBOL":
		return "HENTBOL"
	default:
		return v
	}
}

func (s *TournamentService) ListVenues(sportFilter string) ([]models.Venue, error) {
	rows, err := s.db.Conn.Query(`
		SELECT id, name, COALESCE(address, ''), COALESCE(supported_sports, '[]')
		FROM venues
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wantSport := normalizeSportCode(sportFilter)
	var venues []models.Venue
	for rows.Next() {
		var v models.Venue
		var sportsRaw string
		if err := rows.Scan(&v.ID, &v.Name, &v.Address, &sportsRaw); err != nil {
			return nil, err
		}
		var parsed []string
		_ = json.Unmarshal([]byte(sportsRaw), &parsed)
		if parsed == nil {
			parsed = []string{}
		}
		v.SupportedSports = parsed

		if wantSport != "" {
			matched := false
			for _, sp := range v.SupportedSports {
				if normalizeSportCode(sp) == wantSport {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		venues = append(venues, v)
	}
	if venues == nil {
		venues = []models.Venue{}
	}
	return venues, rows.Err()
}

func (s *TournamentService) CreateVenue(name, address string, supportedSports []string) (*models.Venue, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("venue name is required")
	}
	address = strings.TrimSpace(address)
	normalized := make([]string, 0, len(supportedSports))
	seen := make(map[string]bool)
	for _, sp := range supportedSports {
		code := normalizeSportCode(sp)
		if code != "" && !seen[code] {
			seen[code] = true
			normalized = append(normalized, code)
		}
	}
	sportsJSON, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	res, err := s.db.Conn.Exec(`
		INSERT INTO venues (name, address, supported_sports)
		VALUES (?, ?, ?)`,
		name, address, string(sportsJSON),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.Venue{
		ID:              id,
		Name:            name,
		Address:         address,
		SupportedSports: normalized,
	}, nil
}

func (s *TournamentService) UpdateVenue(id int64, name, address string, supportedSports []string) (*models.Venue, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("venue name is required")
	}
	address = strings.TrimSpace(address)
	normalized := make([]string, 0, len(supportedSports))
	seen := make(map[string]bool)
	for _, sp := range supportedSports {
		code := normalizeSportCode(sp)
		if code != "" && !seen[code] {
			seen[code] = true
			normalized = append(normalized, code)
		}
	}
	sportsJSON, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Conn.Exec(`
		UPDATE venues SET name = ?, address = ?, supported_sports = ? WHERE id = ?`,
		name, address, string(sportsJSON), id,
	); err != nil {
		return nil, err
	}
	return &models.Venue{
		ID:              id,
		Name:            name,
		Address:         address,
		SupportedSports: normalized,
	}, nil
}

func (s *TournamentService) DeleteVenue(id int64) error {
	_, err := s.db.Conn.Exec(`DELETE FROM venues WHERE id = ?`, id)
	return err
}

func (s *TournamentService) ListCategoriesByTournament(tournamentID int64) ([]models.Category, error) {
	rows, err := s.db.Conn.Query(`
		SELECT id, tournament_id, sport_slug, name, COALESCE(format, 'groups'), COALESCE(status, 'AWAITING_DRAW'), created_at
		FROM categories
		WHERE tournament_id = ?
		ORDER BY id ASC`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		var slug string
		if err := rows.Scan(&c.ID, &c.TournamentID, &slug, &c.Name, &c.Format, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.SportSlug = models.SportSlug(slug)
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []models.Category{}
	}
	return categories, rows.Err()
}

func (s *TournamentService) CreateCategory(tournamentID int64, sportSlug models.SportSlug, name string, formatArg ...string) (*models.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("category name is required")
	}
	if !isValidSportSlug(sportSlug) {
		sportSlug = models.SportFootball
	}

	format := "groups"
	if len(formatArg) > 0 {
		rawFmt := strings.ToLower(strings.TrimSpace(formatArg[0]))
		if rawFmt == "knockout" || rawFmt == "eleme" {
			format = "knockout"
		}
	}

	res, err := s.db.Conn.Exec(`
		INSERT INTO categories (tournament_id, sport_slug, name, format, status)
		VALUES (?, ?, ?, ?, 'AWAITING_DRAW')`,
		tournamentID, string(sportSlug), name, format,
	)
	if err != nil {
		return nil, err
	}
	catID, _ := res.LastInsertId()

	return &models.Category{
		ID:           catID,
		TournamentID: tournamentID,
		SportSlug:    sportSlug,
		Name:         name,
		Format:       format,
		Status:       "AWAITING_DRAW",
		CreatedAt:    time.Now(),
	}, nil
}

func (s *TournamentService) DeleteCategory(id int64) error {
	_, err := s.db.Conn.Exec(`DELETE FROM categories WHERE id = ?`, id)
	return err
}

func (s *TournamentService) UpdateCategory(id int64, sportSlug models.SportSlug, name string, formatArg string) (*models.CategoryDetail, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("category name is required")
	}
	if !isValidSportSlug(sportSlug) {
		sportSlug = models.SportFootball
	}
	format := "groups"
	rawFmt := strings.ToLower(strings.TrimSpace(formatArg))
	if rawFmt == "knockout" || rawFmt == "eleme" {
		format = "knockout"
	}
	if _, err := s.db.Conn.Exec(
		`UPDATE categories SET sport_slug = ?, name = ?, format = ? WHERE id = ?`,
		string(sportSlug), name, format, id,
	); err != nil {
		return nil, err
	}
	return s.GetCategoryDetail(id)
}

func (s *TournamentService) CreateStage(categoryID int64, name string, doubleRoundRobin bool) (*models.Stage, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("stage name is required")
	}
	drr := 0
	if doubleRoundRobin {
		drr = 1
	}
	res, err := s.db.Conn.Exec(`
		INSERT INTO stages (category_id, name, stage_type, round_number, double_round_robin)
		VALUES (?, ?, 'group', 1, ?)`,
		categoryID, name, drr,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.Stage{
		ID:               id,
		CategoryID:       categoryID,
		Name:             name,
		StageType:        "group",
		RoundNumber:      1,
		DoubleRoundRobin: doubleRoundRobin,
		CreatedAt:        time.Now(),
	}, nil
}

func (s *TournamentService) DeleteStage(id int64) error {
	tx, err := s.db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE teams SET stage_id = NULL WHERE stage_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM stage_teams WHERE stage_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM stages WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateTeam registers a team into a Category without forcing a group (stageID is optional / nullable).
func (s *TournamentService) CreateTeam(categoryID int64, stageID *int64, schoolName string, _ ...string) (*models.Team, error) {
	schoolName = strings.TrimSpace(schoolName)
	if schoolName == "" {
		return nil, errors.New("school_name is required")
	}

	var stageArg interface{}
	if stageID != nil && *stageID > 0 {
		stageArg = *stageID
	} else {
		stageID = nil
		stageArg = nil
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO teams (category_id, stage_id, school_name, is_seeded)
		VALUES (?, ?, ?, 0)`,
		categoryID, stageArg, schoolName,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	if stageID != nil && *stageID > 0 {
		var maxSlot int
		_ = tx.QueryRow(`SELECT COALESCE(MAX(slot_index), 0) FROM stage_teams WHERE stage_id = ?`, *stageID).Scan(&maxSlot)
		slotIdx := maxSlot + 1
		if _, err := tx.Exec(`INSERT OR REPLACE INTO stage_teams (stage_id, team_id, slot_index) VALUES (?, ?, ?)`, *stageID, id, slotIdx); err != nil {
			return nil, err
		}
		_, _ = tx.Exec(`UPDATE teams SET group_slot_index = ? WHERE id = ?`, slotIdx, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.Team{
		ID:          id,
		CategoryID:  categoryID,
		StageID:     stageID,
		SchoolName:  schoolName,
		DisplayName: models.FormatTeamDisplayName(schoolName),
		IsSeeded:    false,
		CreatedAt:   time.Now(),
	}, nil
}

// UpdateTeamSeed toggles or sets the is_seeded (Seri Başı) flag on a team.
func (s *TournamentService) UpdateTeamSeed(teamID int64, isSeeded bool) error {
	seedInt := 0
	if isSeeded {
		seedInt = 1
	}
	_, err := s.db.Conn.Exec(`UPDATE teams SET is_seeded = ? WHERE id = ?`, seedInt, teamID)
	return err
}

// UpdateTeamStage assigns or unassigns a team to a specific stage/group.
func (s *TournamentService) UpdateTeamStage(teamID int64, stageID *int64) error {
	tx, err := s.db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if stageID == nil || *stageID <= 0 {
		var prevStageID sql.NullInt64
		_ = tx.QueryRow(`SELECT stage_id FROM teams WHERE id = ?`, teamID).Scan(&prevStageID)
		if prevStageID.Valid {
			_, _ = tx.Exec(`DELETE FROM stage_teams WHERE stage_id = ? AND team_id = ?`, prevStageID.Int64, teamID)
		}
		if _, err := tx.Exec(`UPDATE teams SET stage_id = NULL WHERE id = ?`, teamID); err != nil {
			return err
		}
		return tx.Commit()
	}

	// Determine the round_number of the target stage so we replace membership within the same round
	var targetRound int
	var catID int64
	if err := tx.QueryRow(`SELECT category_id, round_number FROM stages WHERE id = ?`, *stageID).Scan(&catID, &targetRound); err == nil {
		_, _ = tx.Exec(`
			DELETE FROM stage_teams
			WHERE team_id = ? AND stage_id IN (
				SELECT id FROM stages WHERE category_id = ? AND round_number = ?
			)`, teamID, catID, targetRound)
	}

	var maxSlot int
	_ = tx.QueryRow(`SELECT COALESCE(MAX(slot_index), 0) FROM stage_teams WHERE stage_id = ?`, *stageID).Scan(&maxSlot)
	slotIdx := maxSlot + 1

	if _, err := tx.Exec(`INSERT OR REPLACE INTO stage_teams (stage_id, team_id, slot_index) VALUES (?, ?, ?)`, *stageID, teamID, slotIdx); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE teams SET stage_id = ?, group_slot_index = ? WHERE id = ?`, *stageID, slotIdx, teamID); err != nil {
		return err
	}

	return tx.Commit()
}

// AutoDistributeTeams distributes all teams in a category evenly across its stages (groups).
func (s *TournamentService) AutoDistributeTeams(categoryID int64) error {
	stages, err := s.listStagesByCategory(categoryID)
	if err != nil {
		return err
	}
	var groupStages []models.Stage
	for _, st := range stages {
		if st.StageType == "group" && st.RoundNumber == 1 {
			groupStages = append(groupStages, st)
		}
	}
	if len(groupStages) == 0 {
		groupStages = stages
	}
	if len(groupStages) == 0 {
		return errors.New("no stages available in this category")
	}
	teams, err := s.listTeamsByCategory(categoryID)
	if err != nil {
		return err
	}
	if len(teams) == 0 {
		return nil
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, st := range groupStages {
		if _, err := tx.Exec(`DELETE FROM stage_teams WHERE stage_id = ?`, st.ID); err != nil {
			return err
		}
	}

	slotCounts := make(map[int64]int)
	for i, tm := range teams {
		targetStageID := groupStages[i%len(groupStages)].ID
		slotCounts[targetStageID]++
		slotIdx := slotCounts[targetStageID]
		if _, err := tx.Exec(`UPDATE teams SET stage_id = ?, group_slot_index = ? WHERE id = ?`, targetStageID, slotIdx, tm.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO stage_teams (stage_id, team_id, slot_index) VALUES (?, ?, ?)`, targetStageID, tm.ID, slotIdx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *TournamentService) DeleteTeam(id int64) error {
	tx, err := s.db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM matches WHERE home_team_id = ? OR away_team_id = ?`, id, id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM teams WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ============================================================================
// MULTI-STAGE PROGRESSION, SEEDED DRAW, POTS & FINAL FOUR WORKFLOW
// ============================================================================

type InitialDrawInput struct {
	Format           string  `json:"format"` // "group" (default) or "knockout"
	SeededTeamIDs    []int64 `json:"seeded_team_ids"`
	GenerateFixtures bool    `json:"generate_fixtures"`
	DoubleRoundRobin bool    `json:"double_round_robin"`
	DefaultVenue     string  `json:"default_venue"`
	DefaultCourt     string  `json:"default_court"`
	StartDate        string  `json:"start_date"`
}

type KnockoutStageInput struct {
	TeamIDs       []int64 `json:"team_ids"`
	SeededTeamIDs []int64 `json:"seeded_team_ids"`
	StageName     string  `json:"stage_name"`
	RoundNumber   int     `json:"round_number"`
	Randomize     bool    `json:"randomize"`
	ResetCategory bool    `json:"reset_category"`
	DefaultVenue  string  `json:"default_venue"`
	DefaultCourt  string  `json:"default_court"`
	StartDate     string  `json:"start_date"`
}

// BuildCategoryKnockout creates a single-elimination knockout stage for 4 and multiples of 4 (4, 8, 16, 32 teams).
func (s *TournamentService) BuildCategoryKnockout(categoryID int64, input KnockoutStageInput) (*models.CategoryDetail, error) {
	allTeams, err := s.listTeamsByCategory(categoryID)
	if err != nil {
		return nil, err
	}
	teamByID := make(map[int64]models.Team, len(allTeams))
	for _, t := range allTeams {
		teamByID[t.ID] = t
	}

	var selectedTeams []models.Team
	if len(input.TeamIDs) > 0 {
		seen := make(map[int64]bool, len(input.TeamIDs))
		for _, tid := range input.TeamIDs {
			if seen[tid] {
				continue
			}
			if tm, ok := teamByID[tid]; ok {
				seen[tid] = true
				selectedTeams = append(selectedTeams, tm)
			}
		}
	} else {
		selectedTeams = allTeams
	}

	if len(selectedTeams) < 2 {
		return nil, fmt.Errorf("eleme (knockout) sistemi için en az 2 takım gereklidir; mevcut: %d", len(selectedTeams))
	}

	roundNum := input.RoundNumber
	if roundNum <= 0 {
		roundNum = 1
	}

	bracketSize, _ := engine.CalculateBracketSize(len(selectedTeams))
	stageName := strings.TrimSpace(input.StageName)
	if stageName == "" {
		stageName = fmt.Sprintf("Eleme Ağacı (%d Takım / %d'li Şema)", len(selectedTeams), bracketSize)
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if len(input.SeededTeamIDs) > 0 {
		if _, err := tx.Exec(`UPDATE teams SET is_seeded = 0 WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
		for _, sid := range input.SeededTeamIDs {
			if _, err := tx.Exec(`UPDATE teams SET is_seeded = 1 WHERE id = ? AND category_id = ?`, sid, categoryID); err != nil {
				return nil, err
			}
		}
	}

	if input.ResetCategory {
		if _, err := tx.Exec(`UPDATE teams SET stage_id = NULL WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM pot_assignments WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM stages WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
	} else {
		if _, err := tx.Exec(`DELETE FROM stages WHERE category_id = ? AND stage_type = 'knockout' AND round_number = ?`, categoryID, roundNum); err != nil {
			return nil, err
		}
	}

	res, err := tx.Exec(`
		INSERT INTO stages (category_id, name, stage_type, round_number, double_round_robin, status)
		VALUES (?, ?, 'knockout', ?, 0, 'ACTIVE')`,
		categoryID, stageName, roundNum,
	)
	if err != nil {
		return nil, err
	}
	stgID, _ := res.LastInsertId()

	if roundNum == 1 && input.ResetCategory {
		_, _ = tx.Exec(`UPDATE categories SET format = 'knockout', status = 'ACTIVE' WHERE id = ?`, categoryID)
	} else {
		_, _ = tx.Exec(`UPDATE categories SET status = 'ACTIVE' WHERE id = ?`, categoryID)
	}

	bracket, err := engine.GenerateKnockoutBracket(categoryID, stgID, selectedTeams, input.SeededTeamIDs, input.Randomize)
	if err != nil {
		return nil, err
	}

	for idx, tm := range bracket.Teams {
		slotIdx := idx + 1
		if _, err := tx.Exec(`UPDATE teams SET stage_id = ?, group_slot_index = ? WHERE id = ?`, stgID, slotIdx, tm.ID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO stage_teams (stage_id, team_id, slot_index) VALUES (?, ?, ?)`, stgID, tm.ID, slotIdx); err != nil {
			return nil, err
		}
	}

	if err := insertStageMatchesTx(tx, bracket.Matches, input.DefaultVenue, input.DefaultCourt, input.StartDate); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetCategoryDetail(categoryID)
}

// ExecuteInitialSeededDraw creates 4-team Stage 1 groups ensuring the selected seeded teams (Seri Başı)
// are isolated in distinct groups first, or builds a direct single-elimination knockout bracket when input.Format == "knockout".
func (s *TournamentService) ExecuteInitialSeededDraw(categoryID int64, input InitialDrawInput) (*models.CategoryDetail, error) {
	if strings.EqualFold(strings.TrimSpace(input.Format), "knockout") {
		return s.BuildCategoryKnockout(categoryID, KnockoutStageInput{
			SeededTeamIDs: input.SeededTeamIDs,
			RoundNumber:   1,
			Randomize:     true,
			ResetCategory: true,
			DefaultVenue:  input.DefaultVenue,
			DefaultCourt:  input.DefaultCourt,
			StartDate:     input.StartDate,
		})
	}
	teams, err := s.listTeamsByCategory(categoryID)
	if err != nil {
		return nil, err
	}
	if len(teams) < 2 {
		return nil, errors.New("at least 2 teams are required in the category for the initial draw")
	}

	allTeamIDs := make([]int, 0, len(teams))
	teamByID := make(map[int64]models.Team, len(teams))
	for _, t := range teams {
		allTeamIDs = append(allTeamIDs, int(t.ID))
		teamByID[t.ID] = t
	}

	seededInts := make([]int, 0, len(input.SeededTeamIDs))
	if len(input.SeededTeamIDs) > 0 {
		for _, sid := range input.SeededTeamIDs {
			seededInts = append(seededInts, int(sid))
		}
	} else {
		// Fall back to teams already marked is_seeded in DB
		for _, t := range teams {
			if t.IsSeeded {
				seededInts = append(seededInts, int(t.ID))
			}
		}
	}

	drawGroups, err := engine.GenerateInitialGroups(int(categoryID), seededInts, allTeamIDs)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Update is_seeded flags on teams
	if _, err := tx.Exec(`UPDATE teams SET is_seeded = 0 WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}
	for _, sid := range seededInts {
		if _, err := tx.Exec(`UPDATE teams SET is_seeded = 1 WHERE id = ? AND category_id = ?`, sid, categoryID); err != nil {
			return nil, err
		}
	}

	// Reset existing stages and matches in this category before creating the fresh Stage 1 draw
	if _, err := tx.Exec(`UPDATE teams SET stage_id = NULL WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM pot_assignments WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM stages WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}
	_, _ = tx.Exec(`UPDATE categories SET format = 'groups', status = 'ACTIVE' WHERE id = ?`, categoryID)

	drr := 0
	if input.DoubleRoundRobin {
		drr = 1
	}

	for _, grp := range drawGroups {
		res, err := tx.Exec(`
			INSERT INTO stages (category_id, name, stage_type, round_number, double_round_robin, status)
			VALUES (?, ?, 'group', 1, ?, 'ACTIVE')`,
			categoryID, grp.GroupName, drr,
		)
		if err != nil {
			return nil, err
		}
		stgID, _ := res.LastInsertId()

		groupTeamModels := make([]models.Team, 0, len(grp.TeamIDs))
		for idx, tidInt := range grp.TeamIDs {
			tid := int64(tidInt)
			slotIdx := idx + 1 // Seeded team is at idx 0 -> slot_index = 1
			if _, err := tx.Exec(`UPDATE teams SET stage_id = ?, group_slot_index = ? WHERE id = ?`, stgID, slotIdx, tid); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`INSERT OR REPLACE INTO stage_teams (stage_id, team_id, slot_index) VALUES (?, ?, ?)`, stgID, tid, slotIdx); err != nil {
				return nil, err
			}
			if tm, ok := teamByID[tid]; ok {
				tm.SlotIndex = slotIdx
				groupTeamModels = append(groupTeamModels, tm)
			}
		}

		if input.GenerateFixtures && len(groupTeamModels) >= 2 {
			matches := engine.GenerateRoundRobin(categoryID, stgID, groupTeamModels, input.DoubleRoundRobin)
			if err := insertStageMatchesTx(tx, matches, input.DefaultVenue, input.DefaultCourt, input.StartDate); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetCategoryDetail(categoryID)
}

// ResetCategoryFixtures deletes all scheduled and completed matches, generated standings, and bracket nodes
// for a category (or a specific stage if stageID is provided), while preserving all registered teams intact
// and resetting status back to AWAITING_DRAW.
func (s *TournamentService) ResetCategoryFixtures(categoryID int64, stageID *int64) (*models.CategoryDetail, error) {
	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if stageID != nil && *stageID > 0 {
		if _, err := tx.Exec(`DELETE FROM matches WHERE category_id = ? AND stage_id = ?`, categoryID, *stageID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE stages SET status = 'AWAITING_DRAW' WHERE id = ? AND category_id = ?`, *stageID, categoryID); err != nil {
			return nil, err
		}
	} else {
		if _, err := tx.Exec(`DELETE FROM matches WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM pot_assignments WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE teams SET stage_id = NULL, group_slot_index = 0 WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM stages WHERE category_id = ?`, categoryID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE categories SET status = 'AWAITING_DRAW' WHERE id = ?`, categoryID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetCategoryDetail(categoryID)
}

// PrepareCategoryPots populates Pot 1 (group winners) and Pot 2 (group runners-up)
// from the completed stages in the category and persists them in pot_assignments.
func (s *TournamentService) PrepareCategoryPots(categoryID int64, stageIDs []int64) (*models.CategoryDetail, error) {
	detail, err := s.GetCategoryDetail(categoryID)
	if err != nil {
		return nil, err
	}

	var targetStages []models.Stage
	if len(stageIDs) > 0 {
		want := make(map[int64]bool, len(stageIDs))
		for _, id := range stageIDs {
			want[id] = true
		}
		for _, st := range detail.Stages {
			if want[st.ID] && st.StageType == "group" {
				targetStages = append(targetStages, st)
			}
		}
	} else {
		// Automatically find the highest round_number among 'group' stages
		maxGroupRound := 0
		for _, st := range detail.Stages {
			if st.StageType == "group" && st.RoundNumber > maxGroupRound {
				maxGroupRound = st.RoundNumber
			}
		}
		for _, st := range detail.Stages {
			if st.StageType == "group" && st.RoundNumber == maxGroupRound {
				targetStages = append(targetStages, st)
			}
		}
	}

	if len(targetStages) == 0 {
		return nil, errors.New("no group stages found to prepare pots from")
	}

	completedStageIDs := make([]int, 0, len(targetStages))
	standingsMap := make(map[int][]models.Standing, len(targetStages))
	maxRound := 1
	for _, st := range targetStages {
		completedStageIDs = append(completedStageIDs, int(st.ID))
		standingsMap[int(st.ID)] = detail.Standings[st.ID]
		if st.RoundNumber > maxRound {
			maxRound = st.RoundNumber
		}
	}

	potRes, err := engine.PrepareNextRoundPots(completedStageIDs, standingsMap)
	if err != nil {
		return nil, err
	}

	nextRound := maxRound + 1

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM pot_assignments WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}

	for _, entry := range potRes.Entries {
		var stgArg interface{}
		if entry.StageID != nil && *entry.StageID > 0 {
			stgArg = *entry.StageID
		}
		if _, err := tx.Exec(`
			INSERT INTO pot_assignments (category_id, round_number, stage_id, team_id, pot_number)
			VALUES (?, ?, ?, ?, ?)`,
			categoryID, nextRound, stgArg, entry.TeamID, entry.PotNumber,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetCategoryDetail(categoryID)
}

type PotEntryInput struct {
	TeamID  int64  `json:"team_id"`
	StageID *int64 `json:"stage_id,omitempty"`
}

type SavePotsInput struct {
	RoundNumber int             `json:"round_number"`
	Pot1        []PotEntryInput `json:"pot1"`
	Pot2        []PotEntryInput `json:"pot2"`
	Pot1TeamIDs []int64         `json:"pot1_team_ids"`
	Pot2TeamIDs []int64         `json:"pot2_team_ids"`
}

// SaveCategoryPots persists manual organizer overrides to Pot 1 and Pot 2 (drag & drop swaps, wild-card additions, removals).
func (s *TournamentService) SaveCategoryPots(categoryID int64, input SavePotsInput) (*models.CategoryDetail, error) {
	// Resolve existing pot assignments so teams swapped between Pot 1 and Pot 2 preserve their source stage_id,
	// while newly added wild-card / bypass teams default to stage_id = nil unless explicitly specified.
	existingPots, err := s.listPotAssignmentsByCategory(categoryID)
	if err != nil {
		return nil, err
	}
	existingPotStage := make(map[int64]*int64, len(existingPots))
	for _, p := range existingPots {
		if p.StageID != nil && *p.StageID > 0 {
			v := *p.StageID
			existingPotStage[p.TeamID] = &v
		}
	}

	_, latestRound, err := s.resolvePreviousGroupHistory(categoryID)
	if err != nil {
		return nil, err
	}

	roundNumber := input.RoundNumber
	if roundNumber <= 0 {
		roundNumber = latestRound + 1
	}

	pot1Entries := input.Pot1
	if len(pot1Entries) == 0 && len(input.Pot1TeamIDs) > 0 {
		for _, tid := range input.Pot1TeamIDs {
			pot1Entries = append(pot1Entries, PotEntryInput{TeamID: tid, StageID: existingPotStage[tid]})
		}
	}

	pot2Entries := input.Pot2
	if len(pot2Entries) == 0 && len(input.Pot2TeamIDs) > 0 {
		for _, tid := range input.Pot2TeamIDs {
			pot2Entries = append(pot2Entries, PotEntryInput{TeamID: tid, StageID: existingPotStage[tid]})
		}
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM pot_assignments WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}

	seen := make(map[int64]bool)
	insertPotList := func(entries []PotEntryInput, potNum int) error {
		for _, e := range entries {
			if e.TeamID <= 0 || seen[e.TeamID] {
				continue
			}
			seen[e.TeamID] = true
			var stgArg interface{}
			if e.StageID != nil && *e.StageID > 0 {
				stgArg = *e.StageID
			} else if prevStg, ok := existingPotStage[e.TeamID]; ok && prevStg != nil && *prevStg > 0 {
				stgArg = *prevStg
			}
			if _, err := tx.Exec(`
				INSERT INTO pot_assignments (category_id, round_number, stage_id, team_id, pot_number)
				VALUES (?, ?, ?, ?, ?)`,
				categoryID, roundNumber, stgArg, e.TeamID, potNum,
			); err != nil {
				return err
			}
		}
		return nil
	}

	if err := insertPotList(pot1Entries, 1); err != nil {
		return nil, err
	}
	if err := insertPotList(pot2Entries, 2); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetCategoryDetail(categoryID)
}

type NextRoundDrawInput struct {
	Format           string  `json:"format"` // "group" (default) or "knockout"
	Pot1TeamIDs      []int64 `json:"pot1_team_ids"`
	Pot2TeamIDs      []int64 `json:"pot2_team_ids"`
	GenerateFixtures bool    `json:"generate_fixtures"`
	DoubleRoundRobin bool    `json:"double_round_robin"`
	DefaultVenue     string  `json:"default_venue"`
	DefaultCourt     string  `json:"default_court"`
	StartDate        string  `json:"start_date"`
}

// ExecuteCategoryNextRoundDraw runs the constraint-based draw from Pot 1 and Pot 2,
// creating the next round's 4-team groups and ensuring teams from the same previous group do not clash.
func (s *TournamentService) ExecuteCategoryNextRoundDraw(categoryID int64, input NextRoundDrawInput) (*models.CategoryDetail, error) {
	teams, err := s.listTeamsByCategory(categoryID)
	if err != nil {
		return nil, err
	}
	teamByID := make(map[int64]models.Team, len(teams))
	for _, t := range teams {
		teamByID[t.ID] = t
	}

	prevHistory, latestRound, err := s.resolvePreviousGroupHistory(categoryID)
	if err != nil {
		return nil, err
	}

	var pot1 []int
	var pot2 []int

	if len(input.Pot1TeamIDs) > 0 || len(input.Pot2TeamIDs) > 0 {
		for _, id := range input.Pot1TeamIDs {
			pot1 = append(pot1, int(id))
		}
		for _, id := range input.Pot2TeamIDs {
			pot2 = append(pot2, int(id))
		}
	} else {
		pots, err := s.listPotAssignmentsByCategory(categoryID)
		if err != nil {
			return nil, err
		}
		for _, p := range pots {
			if p.StageID != nil && *p.StageID > 0 {
				prevHistory[int(p.TeamID)] = int(*p.StageID)
			} else {
				delete(prevHistory, int(p.TeamID))
			}
			if p.PotNumber == 1 {
				pot1 = append(pot1, int(p.TeamID))
			} else if p.PotNumber == 2 {
				pot2 = append(pot2, int(p.TeamID))
			}
		}
	}

	if len(pot1)+len(pot2) < 2 {
		return nil, errors.New("pots are empty; please prepare or add teams to Pot 1 and Pot 2 before drawing")
	}

	nextRound := latestRound + 1
	if nextRound < 2 {
		nextRound = 2
	}

	if strings.EqualFold(strings.TrimSpace(input.Format), "knockout") {
		var combinedIDs []int64
		var seedIDs []int64
		for _, id := range pot1 {
			combinedIDs = append(combinedIDs, int64(id))
			seedIDs = append(seedIDs, int64(id))
		}
		for _, id := range pot2 {
			combinedIDs = append(combinedIDs, int64(id))
		}
		if _, err := s.BuildCategoryKnockout(categoryID, KnockoutStageInput{
			TeamIDs:       combinedIDs,
			SeededTeamIDs: seedIDs,
			StageName:     fmt.Sprintf("%d. Tur - Eleme Ağacı (%d Takım)", nextRound, len(combinedIDs)),
			RoundNumber:   nextRound,
			Randomize:     true,
			ResetCategory: false,
			DefaultVenue:  input.DefaultVenue,
			DefaultCourt:  input.DefaultCourt,
			StartDate:     input.StartDate,
		}); err != nil {
			return nil, err
		}
		_, _ = s.db.Conn.Exec(`DELETE FROM pot_assignments WHERE category_id = ?`, categoryID)
		return s.GetCategoryDetail(categoryID)
	}

	drawnGroups, err := engine.ExecuteNextRoundDraw(int(categoryID), pot1, pot2, prevHistory)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	drr := 0
	if input.DoubleRoundRobin {
		drr = 1
	}

	for _, grp := range drawnGroups {
		stageName := fmt.Sprintf("%d. Tur - %s", nextRound, grp.GroupName)
		res, err := tx.Exec(`
			INSERT INTO stages (category_id, name, stage_type, round_number, double_round_robin)
			VALUES (?, ?, 'group', ?, ?)`,
			categoryID, stageName, nextRound, drr,
		)
		if err != nil {
			return nil, err
		}
		stgID, _ := res.LastInsertId()

		groupTeamModels := make([]models.Team, 0, len(grp.TeamIDs))
		for idx, tidInt := range grp.TeamIDs {
			tid := int64(tidInt)
			slotIdx := idx + 1 // Seeded team is at idx 0 -> slot_index = 1
			if _, err := tx.Exec(`UPDATE teams SET stage_id = ?, group_slot_index = ? WHERE id = ?`, stgID, slotIdx, tid); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`INSERT OR REPLACE INTO stage_teams (stage_id, team_id, slot_index) VALUES (?, ?, ?)`, stgID, tid, slotIdx); err != nil {
				return nil, err
			}
			if tm, ok := teamByID[tid]; ok {
				tm.SlotIndex = slotIdx
				groupTeamModels = append(groupTeamModels, tm)
			}
		}

		if input.GenerateFixtures && len(groupTeamModels) >= 2 {
			matches := engine.GenerateRoundRobin(categoryID, stgID, groupTeamModels, input.DoubleRoundRobin)
			if err := insertStageMatchesTx(tx, matches, input.DefaultVenue, input.DefaultCourt, input.StartDate); err != nil {
				return nil, err
			}
		}
	}

	// Clear pot assignments after draw is finalized
	if _, err := tx.Exec(`DELETE FROM pot_assignments WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetCategoryDetail(categoryID)
}

type FinalFourInput struct {
	GroupAStageID int64  `json:"group_a_stage_id"`
	GroupBStageID int64  `json:"group_b_stage_id"`
	DefaultVenue  string `json:"default_venue"`
	DefaultCourt  string `json:"default_court"`
	StartDate     string `json:"start_date"`
}

// BuildCategoryFinalFour creates the Final Four stage and instantiates Semi-Final 1 (A1 vs B2),
// Semi-Final 2 (B1 vs A2), 3rd Place Match, and Final (Championship Match).
func (s *TournamentService) BuildCategoryFinalFour(categoryID int64, input FinalFourInput) (*models.CategoryDetail, error) {
	detail, err := s.GetCategoryDetail(categoryID)
	if err != nil {
		return nil, err
	}

	groupAID := input.GroupAStageID
	groupBID := input.GroupBStageID
	maxGroupRound := 1

	for _, st := range detail.Stages {
		if st.StageType == "group" && st.RoundNumber > maxGroupRound {
			maxGroupRound = st.RoundNumber
		}
	}

	if groupAID <= 0 || groupBID <= 0 {
		var latestGroups []models.Stage
		for _, st := range detail.Stages {
			if st.StageType == "group" && st.RoundNumber == maxGroupRound {
				latestGroups = append(latestGroups, st)
			}
		}
		if len(latestGroups) < 2 {
			// Fallback to any 2 group stages
			latestGroups = nil
			for _, st := range detail.Stages {
				if st.StageType == "group" {
					latestGroups = append(latestGroups, st)
				}
			}
		}
		if len(latestGroups) < 2 {
			return nil, errors.New("at least 2 group stages (Grup A and Grup B) are required to build the Final Four bracket")
		}
		groupAID = latestGroups[len(latestGroups)-2].ID
		groupBID = latestGroups[len(latestGroups)-1].ID
	}

	standingsMap := map[int][]models.Standing{
		int(groupAID): detail.Standings[groupAID],
		int(groupBID): detail.Standings[groupBID],
	}

	bracket, err := engine.BuildFinalFour(int(groupAID), int(groupBID), standingsMap)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Remove any existing final_four stage for this category before creating the fresh one
	if _, err := tx.Exec(`DELETE FROM stages WHERE category_id = ? AND stage_type = 'final_four'`, categoryID); err != nil {
		return nil, err
	}

	ffRound := maxGroupRound + 1
	res, err := tx.Exec(`
		INSERT INTO stages (category_id, name, stage_type, round_number, parent_stage_id, double_round_robin)
		VALUES (?, 'Final Four', 'final_four', ?, ?, 0)`,
		categoryID, ffRound, groupAID,
	)
	if err != nil {
		return nil, err
	}
	ffStageID, _ := res.LastInsertId()

	for _, qt := range bracket.QualifiedTeams {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO stage_teams (stage_id, team_id) VALUES (?, ?)`, ffStageID, qt.ID); err != nil {
			return nil, err
		}
	}

	for i := range bracket.Matches {
		bracket.Matches[i].CategoryID = categoryID
		bracket.Matches[i].StageID = ffStageID
	}

	if err := insertStageMatchesTx(tx, bracket.Matches, input.DefaultVenue, input.DefaultCourt, input.StartDate); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetCategoryDetail(categoryID)
}

func (s *TournamentService) resolvePreviousGroupHistory(categoryID int64) (map[int]int, int, error) {
	history := make(map[int]int)

	// Find highest round_number among group stages
	var maxRound sql.NullInt64
	_ = s.db.Conn.QueryRow(`
		SELECT MAX(round_number) FROM stages WHERE category_id = ? AND stage_type = 'group'`,
		categoryID,
	).Scan(&maxRound)

	latestRound := 1
	if maxRound.Valid && maxRound.Int64 > 0 {
		latestRound = int(maxRound.Int64)
	}

	// Load stage_teams from the latest group round
	rows, err := s.db.Conn.Query(`
		SELECT st.team_id, st.stage_id
		FROM stage_teams st
		JOIN stages s ON s.id = st.stage_id
		WHERE s.category_id = ? AND s.stage_type = 'group' AND s.round_number = ?`,
		categoryID, latestRound,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tid, sid int
			if err := rows.Scan(&tid, &sid); err == nil {
				history[tid] = sid
			}
		}
	}

	// Also overlay any saved pot_assignments stage_ids
	potRows, err := s.db.Conn.Query(`
		SELECT team_id, stage_id FROM pot_assignments WHERE category_id = ? AND stage_id IS NOT NULL`,
		categoryID,
	)
	if err == nil {
		defer potRows.Close()
		for potRows.Next() {
			var tid, sid int
			if err := potRows.Scan(&tid, &sid); err == nil && sid > 0 {
				history[tid] = sid
			}
		}
	}

	return history, latestRound, nil
}

func insertStageMatchesTx(tx *sql.Tx, matches []models.Match, defaultVenue, defaultCourt, startDate string) error {
	venue := strings.TrimSpace(defaultVenue)
	court := strings.TrimSpace(defaultCourt)
	if venue == "" && len(matches) > 0 && matches[0].CategoryID > 0 {
		var sportSlug string
		if err := tx.QueryRow(`SELECT sport_slug FROM categories WHERE id = ?`, matches[0].CategoryID).Scan(&sportSlug); err == nil {
			wantSport := normalizeSportCode(sportSlug)
			vRows, err := tx.Query(`SELECT name, COALESCE(supported_sports, '[]') FROM venues ORDER BY id ASC`)
			if err == nil {
				for vRows.Next() {
					var vName, vSports string
					if err := vRows.Scan(&vName, &vSports); err == nil {
						var parsed []string
						_ = json.Unmarshal([]byte(vSports), &parsed)
						for _, sp := range parsed {
							if normalizeSportCode(sp) == wantSport {
								venue = vName
								break
							}
						}
					}
					if venue != "" {
						break
					}
				}
				vRows.Close()
			}
		}
	}
	for i := range matches {
		if matches[i].Venue == "" {
			matches[i].Venue = venue
		}
		if matches[i].Court == "" {
			matches[i].Court = court
		}
		if matches[i].ScheduledAt == "" && startDate != "" {
			if baseDate, err := time.Parse("2006-01-02", startDate); err == nil {
				matchDate := baseDate.AddDate(0, 0, matches[i].Round-1)
				hour := 10 + ((matches[i].MatchNumber - 1) % 4) * 2
				matches[i].ScheduledAt = fmt.Sprintf("%s %02d:00", matchDate.Format("2006-01-02"), hour)
			}
		}

		isByeInt := 0
		if matches[i].IsBye {
			isByeInt = 1
		}
		isForfeitInt := 0
		if matches[i].IsForfeit {
			isForfeitInt = 1
		}
		status := matches[i].Status
		if status == "" {
			status = models.StatusScheduled
		}

		res, err := tx.Exec(`
			INSERT INTO matches (
				category_id, stage_id, round, match_number, bracket_slot,
				source_home_stage_id, source_home_rank, source_away_stage_id, source_away_rank,
				home_team_id, away_team_id, home_team_name, away_team_name,
				is_bye, is_forfeit, forfeit_winner_team_id, home_score, away_score,
				home_penalties, away_penalties, next_match_id, next_match_slot, winner_team_id,
				status, venue, court, scheduled_at, notes
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			matches[i].CategoryID, matches[i].StageID, matches[i].Round, matches[i].MatchNumber, matches[i].BracketSlot,
			matches[i].SourceHomeStageID, matches[i].SourceHomeRank, matches[i].SourceAwayStageID, matches[i].SourceAwayRank,
			matches[i].HomeTeamID, matches[i].AwayTeamID, matches[i].HomeTeamName, matches[i].AwayTeamName,
			isByeInt, isForfeitInt, matches[i].ForfeitWinnerTeamID, matches[i].HomeScore, matches[i].AwayScore,
			matches[i].HomePenalties, matches[i].AwayPenalties, matches[i].NextMatchID, matches[i].NextMatchSlot, matches[i].WinnerTeamID,
			string(status), matches[i].Venue, matches[i].Court, matches[i].ScheduledAt, matches[i].Notes,
		)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		matches[i].ID = id
	}

	// Link child bracket matches to their parent match IDs (next_match_id, next_match_slot)
	byRound := make(map[int][]int)
	maxRound := 0
	hasBracketSlots := false
	for i := range matches {
		if matches[i].BracketSlot != "" {
			hasBracketSlots = true
		}
		if matches[i].BracketSlot == string(models.BracketSlotThirdPlace) {
			continue
		}
		r := matches[i].Round
		byRound[r] = append(byRound[r], i)
		if r > maxRound {
			maxRound = r
		}
	}
	if hasBracketSlots && maxRound > 1 {
		for r := 1; r < maxRound; r++ {
			currIndices := byRound[r]
			nextIndices := byRound[r+1]
			for idxInRound, mIdx := range currIndices {
				parentIdx := idxInRound / 2
				if parentIdx < len(nextIndices) {
					parentMatchID := matches[nextIndices[parentIdx]].ID
					nextSlot := "home"
					if idxInRound%2 == 1 {
						nextSlot = "away"
					}
					matches[mIdx].NextMatchID = &parentMatchID
					matches[mIdx].NextMatchSlot = nextSlot
					if _, err := tx.Exec(
						`UPDATE matches SET next_match_id = ?, next_match_slot = ? WHERE id = ?`,
						parentMatchID, nextSlot, matches[mIdx].ID,
					); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

type GenerateFixtureInput struct {
	StageID          int64  `json:"stage_id"`
	DoubleRoundRobin bool   `json:"double_round_robin"`
	DefaultVenue     string `json:"default_venue"`
	DefaultCourt     string `json:"default_court"`
	StartDate        string `json:"start_date"`
}

func (s *TournamentService) GenerateStageFixtures(categoryID int64, input GenerateFixtureInput) ([]models.Match, error) {
	teams, err := s.listTeamsByStage(input.StageID)
	if err != nil {
		return nil, err
	}

	if len(teams) == 0 {
		allCatTeams, err := s.listTeamsByCategory(categoryID)
		if err != nil {
			return nil, err
		}
		var unassigned []models.Team
		for _, t := range allCatTeams {
			if t.StageID == nil {
				unassigned = append(unassigned, t)
			}
		}
		if len(unassigned) >= 2 {
			for _, ut := range unassigned {
				_ = s.UpdateTeamStage(ut.ID, &input.StageID)
			}
			teams, err = s.listTeamsByStage(input.StageID)
			if err != nil {
				return nil, err
			}
		}
	}

	if len(teams) < 2 {
		return nil, errors.New("at least 2 teams in this group are required to generate fixtures")
	}

	matches := engine.GenerateRoundRobin(categoryID, input.StageID, teams, input.DoubleRoundRobin)

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	drr := 0
	if input.DoubleRoundRobin {
		drr = 1
	}
	if _, err := tx.Exec(`UPDATE stages SET double_round_robin = ? WHERE id = ? AND category_id = ?`, drr, input.StageID, categoryID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(`DELETE FROM matches WHERE stage_id = ? AND category_id = ?`, input.StageID, categoryID); err != nil {
		return nil, err
	}

	if err := insertStageMatchesTx(tx, matches, input.DefaultVenue, input.DefaultCourt, input.StartDate); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return matches, nil
}

type UpdateMatchInput struct {
	HomeTeamID          *int64             `json:"home_team_id,omitempty"`
	AwayTeamID          *int64             `json:"away_team_id,omitempty"`
	HomeTeamName        string             `json:"home_team_name,omitempty"`
	AwayTeamName        string             `json:"away_team_name,omitempty"`
	Round               *int               `json:"round,omitempty"`
	MatchNumber         *int               `json:"match_number,omitempty"`
	HomeScore           int                `json:"home_score"`
	AwayScore           int                `json:"away_score"`
	HomePenalties       *int               `json:"home_penalties"`
	AwayPenalties       *int               `json:"away_penalties"`
	Status              models.MatchStatus `json:"status"`
	IsForfeit           bool               `json:"is_forfeit"`
	ForfeitWinnerTeamID *int64             `json:"forfeit_winner_team_id"`
	Venue               string             `json:"venue"`
	Court               string             `json:"court"`
	ScheduledAt         string             `json:"scheduled_at"`
	Notes               string             `json:"notes"`
}

type SwapMatchPairingsInput struct {
	MatchAID int64  `json:"match_a_id"`
	SlotA    string `json:"slot_a"` // "home" or "away"
	MatchBID int64  `json:"match_b_id"`
	SlotB    string `json:"slot_b"` // "home" or "away"
}

// SwapMatchPairings allows organizers to swap teams between slots of scheduled matches (or within the same match).
func (s *TournamentService) SwapMatchPairings(categoryID int64, input SwapMatchPairingsInput) (*models.CategoryDetail, error) {
	mA, err := s.GetMatchByID(input.MatchAID)
	if err != nil || mA.CategoryID != categoryID {
		return nil, errors.New("match A not found in category")
	}
	mB, err := s.GetMatchByID(input.MatchBID)
	if err != nil || mB.CategoryID != categoryID {
		return nil, errors.New("match B not found in category")
	}

	slotA := strings.ToLower(strings.TrimSpace(input.SlotA))
	if slotA != "away" {
		slotA = "home"
	}
	slotB := strings.ToLower(strings.TrimSpace(input.SlotB))
	if slotB != "away" {
		slotB = "home"
	}

	getSlot := func(m *models.Match, slot string) (*int64, string) {
		if slot == "away" {
			return m.AwayTeamID, m.AwayTeamName
		}
		return m.HomeTeamID, m.HomeTeamName
	}

	teamAID, teamAName := getSlot(mA, slotA)
	teamBID, teamBName := getSlot(mB, slotB)

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	setSlotTx := func(matchID int64, slot string, teamID *int64, teamName string) error {
		colID := "home_team_id"
		colName := "home_team_name"
		if slot == "away" {
			colID = "away_team_id"
			colName = "away_team_name"
		}
		_, err := tx.Exec(
			fmt.Sprintf(`UPDATE matches SET %s = ?, %s = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, colID, colName),
			teamID, teamName, matchID,
		)
		return err
	}

	if err := setSlotTx(mA.ID, slotA, teamBID, teamBName); err != nil {
		return nil, err
	}
	if err := setSlotTx(mB.ID, slotB, teamAID, teamAName); err != nil {
		return nil, err
	}

	// Re-evaluate BYE status and next-round advancement if either match is in a knockout bracket
	syncByeMatchTx := func(matchID int64) error {
		var hID, aID, nextID sql.NullInt64
		var hName, aName, nextSlot, notes string
		var statusStr string
		if err := tx.QueryRow(`
			SELECT home_team_id, away_team_id, home_team_name, away_team_name, next_match_id, next_match_slot, notes, status
			FROM matches WHERE id = ?`, matchID).Scan(&hID, &aID, &hName, &aName, &nextID, &nextSlot, &notes, &statusStr); err != nil {
			return err
		}
		if statusStr == string(models.StatusCompleted) {
			return nil
		}

		homeIsBye := !hID.Valid && (strings.EqualFold(hName, "BAY") || strings.EqualFold(hName, "BYE"))
		awayIsBye := !aID.Valid && (strings.EqualFold(aName, "BAY") || strings.EqualFold(aName, "BYE"))

		if (homeIsBye || awayIsBye) && (hID.Valid || aID.Valid) {
			var winID int64
			var winName string
			if hID.Valid {
				winID = hID.Int64
				winName = hName
			} else {
				winID = aID.Int64
				winName = aName
			}
			// Normalize so the real team is in Home and BAY is in Away
			if _, err := tx.Exec(`
				UPDATE matches
				SET home_team_id = ?, home_team_name = ?, away_team_id = NULL, away_team_name = 'BAY',
				    is_bye = 1, winner_team_id = ?, status = ?
				WHERE id = ?`, winID, winName, winID, string(models.StatusByeAdvanced), matchID); err != nil {
				return err
			}
			if nextID.Valid && nextID.Int64 > 0 {
				if nextSlot == "away" {
					_, _ = tx.Exec(`UPDATE matches SET away_team_id = ?, away_team_name = ? WHERE id = ?`, winID, winName, nextID.Int64)
				} else {
					_, _ = tx.Exec(`UPDATE matches SET home_team_id = ?, home_team_name = ? WHERE id = ?`, winID, winName, nextID.Int64)
				}
			}
		} else if hID.Valid && aID.Valid {
			if _, err := tx.Exec(`
				UPDATE matches
				SET is_bye = 0, winner_team_id = NULL, status = ?
				WHERE id = ?`, string(models.StatusScheduled), matchID); err != nil {
				return err
			}
			if nextID.Valid && nextID.Int64 > 0 {
				placeholder := fmt.Sprintf("%s Galibi", nonEmptyNote(notes, "Maç"))
				if nextSlot == "away" {
					_, _ = tx.Exec(`UPDATE matches SET away_team_id = NULL, away_team_name = ? WHERE id = ? AND status = 'scheduled'`, placeholder, nextID.Int64)
				} else {
					_, _ = tx.Exec(`UPDATE matches SET home_team_id = NULL, home_team_name = ? WHERE id = ? AND status = 'scheduled'`, placeholder, nextID.Int64)
				}
			}
		}
		return nil
	}

	if err := syncByeMatchTx(mA.ID); err != nil {
		return nil, err
	}
	if mB.ID != mA.ID {
		if err := syncByeMatchTx(mB.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetCategoryDetail(categoryID)
}

func (s *TournamentService) UpdateMatch(matchID int64, input UpdateMatchInput) (*models.Match, error) {
	existing, err := s.GetMatchByID(matchID)
	if err != nil {
		return nil, errors.New("match not found")
	}

	var sportSlugStr string
	_ = s.db.Conn.QueryRow(`SELECT sport_slug FROM categories WHERE id = ?`, existing.CategoryID).Scan(&sportSlugStr)
	sportSlug := models.SportSlug(sportSlugStr)

	var stageType string
	_ = s.db.Conn.QueryRow(`SELECT stage_type FROM stages WHERE id = ?`, existing.StageID).Scan(&stageType)
	isElimination := stageType == "final_four" || stageType == "knockout" || existing.BracketSlot != ""

	// Resolve editable pairings / team assignments when provided
	homeTeamID := existing.HomeTeamID
	homeTeamName := existing.HomeTeamName
	if input.HomeTeamID != nil {
		if *input.HomeTeamID <= 0 {
			homeTeamID = nil
			if strings.TrimSpace(input.HomeTeamName) != "" {
				homeTeamName = strings.TrimSpace(input.HomeTeamName)
			}
		} else {
			tid := *input.HomeTeamID
			homeTeamID = &tid
			var school string
			if err := s.db.Conn.QueryRow(`SELECT school_name FROM teams WHERE id = ?`, tid).Scan(&school); err == nil {
				homeTeamName = models.FormatTeamDisplayName(school)
			} else if strings.TrimSpace(input.HomeTeamName) != "" {
				homeTeamName = strings.TrimSpace(input.HomeTeamName)
			}
		}
	} else if strings.TrimSpace(input.HomeTeamName) != "" {
		homeTeamName = strings.TrimSpace(input.HomeTeamName)
	}

	awayTeamID := existing.AwayTeamID
	awayTeamName := existing.AwayTeamName
	if input.AwayTeamID != nil {
		if *input.AwayTeamID <= 0 {
			awayTeamID = nil
			if strings.TrimSpace(input.AwayTeamName) != "" {
				awayTeamName = strings.TrimSpace(input.AwayTeamName)
			}
		} else {
			tid := *input.AwayTeamID
			awayTeamID = &tid
			var school string
			if err := s.db.Conn.QueryRow(`SELECT school_name FROM teams WHERE id = ?`, tid).Scan(&school); err == nil {
				awayTeamName = models.FormatTeamDisplayName(school)
			} else if strings.TrimSpace(input.AwayTeamName) != "" {
				awayTeamName = strings.TrimSpace(input.AwayTeamName)
			}
		}
	} else if strings.TrimSpace(input.AwayTeamName) != "" {
		awayTeamName = strings.TrimSpace(input.AwayTeamName)
	}

	roundNum := existing.Round
	if input.Round != nil && *input.Round > 0 {
		roundNum = *input.Round
	}
	matchNum := existing.MatchNumber
	if input.MatchNumber != nil && *input.MatchNumber > 0 {
		matchNum = *input.MatchNumber
	}

	status := input.Status
	if status != models.StatusScheduled && status != models.StatusCompleted {
		status = models.StatusCompleted
	}

	homeScore := input.HomeScore
	awayScore := input.AwayScore
	if homeScore < 0 {
		homeScore = 0
	}
	if awayScore < 0 {
		awayScore = 0
	}

	isForfeit := input.IsForfeit
	var forfeitWinnerID *int64
	var homePenalties, awayPenalties *int

	if status == models.StatusScheduled {
		homeScore = 0
		awayScore = 0
		isForfeit = false
		forfeitWinnerID = nil
		homePenalties = nil
		awayPenalties = nil
	} else if isForfeit {
		status = models.StatusCompleted
		winScore, loseScore := scoring.OfficialForfeitScore(sportSlug)

		if input.ForfeitWinnerTeamID != nil {
			if homeTeamID != nil && *input.ForfeitWinnerTeamID == *homeTeamID {
				homeScore = winScore
				awayScore = loseScore
				forfeitWinnerID = homeTeamID
			} else if awayTeamID != nil && *input.ForfeitWinnerTeamID == *awayTeamID {
				homeScore = loseScore
				awayScore = winScore
				forfeitWinnerID = awayTeamID
			}
		}
		if forfeitWinnerID == nil {
			if awayScore > homeScore && awayTeamID != nil {
				homeScore = loseScore
				awayScore = winScore
				forfeitWinnerID = awayTeamID
			} else {
				homeScore = winScore
				awayScore = loseScore
				forfeitWinnerID = homeTeamID
			}
		}
		homePenalties = nil
		awayPenalties = nil
	} else if !existing.IsBye {
		// Completed non-forfeit match: check draw and penalty rules
		if homeScore == awayScore {
			if isElimination {
				// Final Four & Elimination matches CANNOT end in a draw; decisive penalties are mandatory
				if input.HomePenalties == nil || input.AwayPenalties == nil {
					return nil, errors.New("Final Four ve eleme maçları berabere bitemez; kazananı belirlemek için penaltı skorlarını giriniz")
				}
				hp := *input.HomePenalties
				ap := *input.AwayPenalties
				if hp < 0 || ap < 0 || hp == ap {
					return nil, errors.New("Final Four ve eleme maçlarında penaltı skorları eşit olamaz; kesin bir galip belirlenmelidir")
				}
				homePenalties = &hp
				awayPenalties = &ap
			} else if input.HomePenalties != nil && input.AwayPenalties != nil {
				hp := *input.HomePenalties
				ap := *input.AwayPenalties
				if hp < 0 {
					hp = 0
				}
				if ap < 0 {
					ap = 0
				}
				if (hp > 0 || ap > 0) && hp == ap {
					return nil, errors.New("penaltı atışları berabere bitemez; eşit olmayan penaltı skoru giriniz")
				}
				if hp > 0 || ap > 0 {
					homePenalties = &hp
					awayPenalties = &ap
				}
			}
		} else {
			// Regulation/extra time already produced a winner
			homePenalties = nil
			awayPenalties = nil
		}
	}

	isForfeitInt := 0
	if isForfeit {
		isForfeitInt = 1
	}

	tx, err := s.db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE matches
		SET round = ?, match_number = ?,
		    home_team_id = ?, away_team_id = ?, home_team_name = ?, away_team_name = ?,
		    home_score = ?, away_score = ?, home_penalties = ?, away_penalties = ?, status = ?,
		    is_forfeit = ?, forfeit_winner_team_id = ?,
		    venue = ?, court = ?, scheduled_at = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		roundNum, matchNum,
		homeTeamID, awayTeamID, homeTeamName, awayTeamName,
		homeScore, awayScore, homePenalties, awayPenalties, string(status),
		isForfeitInt, forfeitWinnerID,
		strings.TrimSpace(input.Venue), strings.TrimSpace(input.Court),
		strings.TrimSpace(input.ScheduledAt), strings.TrimSpace(input.Notes),
		matchID,
	)
	if err != nil {
		return nil, err
	}

	// Determine decisive winner & loser (via regulation score or penalty shootout)
	hasDecisiveWinner := false
	homeWon := false
	var matchWinnerID *int64
	if status == models.StatusCompleted {
		if homeScore > awayScore {
			hasDecisiveWinner = true
			homeWon = true
			matchWinnerID = homeTeamID
		} else if awayScore > homeScore {
			hasDecisiveWinner = true
			homeWon = false
			matchWinnerID = awayTeamID
		} else if homePenalties != nil && awayPenalties != nil && *homePenalties != *awayPenalties {
			hasDecisiveWinner = true
			homeWon = *homePenalties > *awayPenalties
			if homeWon {
				matchWinnerID = homeTeamID
			} else {
				matchWinnerID = awayTeamID
			}
		}
	}
	_, _ = tx.Exec(`UPDATE matches SET winner_team_id = ? WHERE id = ?`, matchWinnerID, matchID)

	// 1. If this match is Semi-Final 1 or Semi-Final 2, automatically advance Winner to Final and Loser to 3rd Place
	if existing.BracketSlot == string(models.BracketSlotSemiFinal1) || existing.BracketSlot == string(models.BracketSlotSemiFinal2) {
		var winnerID, loserID *int64
		var winnerName, loserName string

		if hasDecisiveWinner {
			if homeWon {
				winnerID = homeTeamID
				winnerName = homeTeamName
				loserID = awayTeamID
				loserName = awayTeamName
			} else {
				winnerID = awayTeamID
				winnerName = awayTeamName
				loserID = homeTeamID
				loserName = homeTeamName
			}
		} else {
			if existing.BracketSlot == string(models.BracketSlotSemiFinal1) {
				winnerName = "Yarı Final 1 Galibi"
				loserName = "Yarı Final 1 Mağlubu"
			} else {
				winnerName = "Yarı Final 2 Galibi"
				loserName = "Yarı Final 2 Mağlubu"
			}
		}

		if existing.BracketSlot == string(models.BracketSlotSemiFinal1) {
			_, _ = tx.Exec(`
				UPDATE matches SET home_team_id = ?, home_team_name = ?, updated_at = CURRENT_TIMESTAMP
				WHERE stage_id = ? AND bracket_slot = ?`,
				winnerID, winnerName, existing.StageID, string(models.BracketSlotFinal),
			)
			_, _ = tx.Exec(`
				UPDATE matches SET home_team_id = ?, home_team_name = ?, updated_at = CURRENT_TIMESTAMP
				WHERE stage_id = ? AND bracket_slot = ?`,
				loserID, loserName, existing.StageID, string(models.BracketSlotThirdPlace),
			)
		} else {
			_, _ = tx.Exec(`
				UPDATE matches SET away_team_id = ?, away_team_name = ?, updated_at = CURRENT_TIMESTAMP
				WHERE stage_id = ? AND bracket_slot = ?`,
				winnerID, winnerName, existing.StageID, string(models.BracketSlotFinal),
			)
			_, _ = tx.Exec(`
				UPDATE matches SET away_team_id = ?, away_team_name = ?, updated_at = CURRENT_TIMESTAMP
				WHERE stage_id = ? AND bracket_slot = ?`,
				loserID, loserName, existing.StageID, string(models.BracketSlotThirdPlace),
			)
		}
	} else if stageType == "knockout" && existing.BracketSlot != string(models.BracketSlotFinal) && existing.BracketSlot != string(models.BracketSlotThirdPlace) {
		// 2. For earlier rounds in a knockout bracket (Round of 32, Round of 16, Quarter-Finals), advance Winner to the next round match slot
		var roundMatchIDs []int64
		rRows, err := tx.Query(`
			SELECT id FROM matches
			WHERE stage_id = ? AND round = ? AND bracket_slot != ?
			ORDER BY match_number ASC, id ASC`,
			existing.StageID, roundNum, string(models.BracketSlotThirdPlace),
		)
		if err == nil {
			for rRows.Next() {
				var mid int64
				if err := rRows.Scan(&mid); err == nil {
					roundMatchIDs = append(roundMatchIDs, mid)
				}
			}
			rRows.Close()
		}

		idxInRound := -1
		for idx, mid := range roundMatchIDs {
			if mid == matchID {
				idxInRound = idx
				break
			}
		}

		if idxInRound >= 0 {
			var nextRoundMatchIDs []int64
			nrRows, err := tx.Query(`
				SELECT id FROM matches
				WHERE stage_id = ? AND round = ? AND bracket_slot != ?
				ORDER BY match_number ASC, id ASC`,
				existing.StageID, roundNum+1, string(models.BracketSlotThirdPlace),
			)
			if err == nil {
				for nrRows.Next() {
					var mid int64
					if err := nrRows.Scan(&mid); err == nil {
						nextRoundMatchIDs = append(nextRoundMatchIDs, mid)
					}
				}
				nrRows.Close()
			}

			targetIdx := idxInRound / 2
			if targetIdx < len(nextRoundMatchIDs) {
				targetMatchID := nextRoundMatchIDs[targetIdx]
				var winnerID *int64
				winnerName := fmt.Sprintf("%s Galibi", nonEmptyNote(existing.Notes, fmt.Sprintf("Maç %d", matchNum)))
				if hasDecisiveWinner {
					if homeWon {
						winnerID = homeTeamID
						winnerName = homeTeamName
					} else {
						winnerID = awayTeamID
						winnerName = awayTeamName
					}
				}
				if idxInRound%2 == 0 {
					_, _ = tx.Exec(`
						UPDATE matches SET home_team_id = ?, home_team_name = ?, updated_at = CURRENT_TIMESTAMP
						WHERE id = ?`,
						winnerID, winnerName, targetMatchID,
					)
				} else {
					_, _ = tx.Exec(`
						UPDATE matches SET away_team_id = ?, away_team_name = ?, updated_at = CURRENT_TIMESTAMP
						WHERE id = ?`,
						winnerID, winnerName, targetMatchID,
					)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetMatchByID(matchID)
}

func nonEmptyNote(note, fallback string) string {
	if strings.TrimSpace(note) != "" {
		return strings.TrimSpace(note)
	}
	return fallback
}

func (s *TournamentService) GetMatchByID(matchID int64) (*models.Match, error) {
	var m models.Match
	var isByeInt, isForfeitInt int
	var statusStr string
	var homeID, awayID, srcHomeStg, srcAwayStg, forfeitWinID, homePen, awayPen, nextMatchID, winnerTeamID sql.NullInt64

	err := s.db.Conn.QueryRow(`
		SELECT m.id, m.category_id, m.stage_id, COALESCE(st.name, ''), m.round, m.match_number,
		       m.bracket_slot, m.source_home_stage_id, m.source_home_rank, m.source_away_stage_id, m.source_away_rank,
		       m.home_team_id, m.away_team_id, m.home_team_name, m.away_team_name,
		       m.is_bye, m.is_forfeit, m.forfeit_winner_team_id, m.home_score, m.away_score,
		       m.home_penalties, m.away_penalties, m.next_match_id, m.next_match_slot, m.winner_team_id,
		       m.status, m.venue, m.court, m.scheduled_at, m.notes, m.updated_at
		FROM matches m
		LEFT JOIN stages st ON st.id = m.stage_id
		WHERE m.id = ?`, matchID).Scan(
		&m.ID, &m.CategoryID, &m.StageID, &m.StageName, &m.Round, &m.MatchNumber,
		&m.BracketSlot, &srcHomeStg, &m.SourceHomeRank, &srcAwayStg, &m.SourceAwayRank,
		&homeID, &awayID, &m.HomeTeamName, &m.AwayTeamName,
		&isByeInt, &isForfeitInt, &forfeitWinID, &m.HomeScore, &m.AwayScore,
		&homePen, &awayPen, &nextMatchID, &m.NextMatchSlot, &winnerTeamID,
		&statusStr, &m.Venue, &m.Court, &m.ScheduledAt, &m.Notes, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if homeID.Valid {
		v := homeID.Int64
		m.HomeTeamID = &v
	}
	if awayID.Valid {
		v := awayID.Int64
		m.AwayTeamID = &v
	}
	if srcHomeStg.Valid {
		v := srcHomeStg.Int64
		m.SourceHomeStageID = &v
	}
	if srcAwayStg.Valid {
		v := srcAwayStg.Int64
		m.SourceAwayStageID = &v
	}
	if forfeitWinID.Valid {
		v := forfeitWinID.Int64
		m.ForfeitWinnerTeamID = &v
	}
	if homePen.Valid {
		v := int(homePen.Int64)
		m.HomePenalties = &v
	}
	if awayPen.Valid {
		v := int(awayPen.Int64)
		m.AwayPenalties = &v
	}
	if nextMatchID.Valid {
		v := nextMatchID.Int64
		m.NextMatchID = &v
	}
	if winnerTeamID.Valid {
		v := winnerTeamID.Int64
		m.WinnerTeamID = &v
	}
	m.IsBye = isByeInt == 1
	m.IsForfeit = isForfeitInt == 1
	m.Status = models.MatchStatus(statusStr)
	return &m, nil
}

func (s *TournamentService) GetCategoryDetail(categoryID int64) (*models.CategoryDetail, error) {
	var cat models.Category
	var sportSlug string
	var tournamentName string

	err := s.db.Conn.QueryRow(`
		SELECT c.id, c.tournament_id, c.sport_slug, c.name, COALESCE(c.format, 'groups'), COALESCE(c.status, 'AWAITING_DRAW'), c.created_at, COALESCE(t.name, '')
		FROM categories c
		LEFT JOIN tournaments t ON t.id = c.tournament_id
		WHERE c.id = ?`, categoryID).Scan(
		&cat.ID, &cat.TournamentID, &sportSlug, &cat.Name, &cat.Format, &cat.Status, &cat.CreatedAt, &tournamentName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	cat.SportSlug = models.SportSlug(sportSlug)

	stages, err := s.listStagesByCategory(categoryID)
	if err != nil {
		return nil, err
	}

	teams, err := s.listTeamsByCategory(categoryID)
	if err != nil {
		return nil, err
	}

	matches, err := s.listMatchesByCategory(categoryID)
	if err != nil {
		return nil, err
	}

	pots, err := s.listPotAssignmentsByCategory(categoryID)
	if err != nil {
		return nil, err
	}

	strategy := scoring.GetStrategy(cat.SportSlug)
	standingsByStage := make(map[int64][]models.Standing)
	var allStandings []models.Standing

	matchesByStage := make(map[int64][]models.Match)
	for _, m := range matches {
		matchesByStage[m.StageID] = append(matchesByStage[m.StageID], m)
	}

	currentRound := 1
	for i := range stages {
		stg := &stages[i]
		if stg.RoundNumber > currentRound {
			currentRound = stg.RoundNumber
		}
		stgTeams, err := s.listTeamsByStage(stg.ID)
		if err != nil {
			return nil, err
		}
		// Seeded status badge ("Seri Başı") only applies to the initial stage (Round 1 group stage)
		if stg.RoundNumber > 1 || stg.StageType != "group" {
			for ti := range stgTeams {
				stgTeams[ti].IsSeeded = false
			}
		}
		stg.Teams = stgTeams
		stg.Matches = matchesByStage[stg.ID]
		if stg.Matches == nil {
			stg.Matches = []models.Match{}
		}
		if stg.StageType != "final_four" && stg.StageType != "knockout" {
			st := strategy.CalculateStandings(stg.ID, stg.Name, stg.Teams, stg.Matches)
			if stg.RoundNumber > 1 {
				for si := range st {
					st[si].IsSeeded = false
				}
			}
			standingsByStage[stg.ID] = st
			allStandings = append(allStandings, st...)
		} else {
			standingsByStage[stg.ID] = []models.Standing{}
		}
	}

	if allStandings == nil {
		allStandings = []models.Standing{}
	}

	return &models.CategoryDetail{
		Category:       cat,
		TournamentName: tournamentName,
		Stages:         stages,
		Teams:          teams,
		Matches:        matches,
		Standings:      standingsByStage,
		AllStandings:   allStandings,
		PotAssignments: pots,
		CurrentRound:   currentRound,
	}, nil
}

func (s *TournamentService) listStagesByCategory(categoryID int64) ([]models.Stage, error) {
	rows, err := s.db.Conn.Query(`
		SELECT id, category_id, name, stage_type, round_number, parent_stage_id, double_round_robin, COALESCE(status, 'AWAITING_DRAW'), created_at
		FROM stages
		WHERE category_id = ?
		ORDER BY round_number ASC, id ASC`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []models.Stage
	for rows.Next() {
		var st models.Stage
		var drr int
		var parentID sql.NullInt64
		if err := rows.Scan(&st.ID, &st.CategoryID, &st.Name, &st.StageType, &st.RoundNumber, &parentID, &drr, &st.Status, &st.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := parentID.Int64
			st.ParentStageID = &v
		}
		st.DoubleRoundRobin = drr == 1
		stages = append(stages, st)
	}
	if stages == nil {
		stages = []models.Stage{}
	}
	return stages, rows.Err()
}

func (s *TournamentService) listTeamsByCategory(categoryID int64) ([]models.Team, error) {
	rows, err := s.db.Conn.Query(`
		SELECT id, category_id, stage_id, group_slot_index, school_name, team_tag, is_seeded, created_at
		FROM teams
		WHERE category_id = ?
		ORDER BY id ASC`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		var stgID sql.NullInt64
		var isSeededInt int
		if err := rows.Scan(&t.ID, &t.CategoryID, &stgID, &t.SlotIndex, &t.SchoolName, &t.TeamTag, &isSeededInt, &t.CreatedAt); err != nil {
			return nil, err
		}
		if stgID.Valid {
			v := stgID.Int64
			t.StageID = &v
		}
		t.IsSeeded = isSeededInt == 1
		t.DisplayName = models.FormatTeamDisplayName(t.SchoolName, t.TeamTag)
		teams = append(teams, t)
	}
	if teams == nil {
		teams = []models.Team{}
	}
	return teams, rows.Err()
}

func (s *TournamentService) listTeamsByStage(stageID int64) ([]models.Team, error) {
	rows, err := s.db.Conn.Query(`
		SELECT t.id, t.category_id, t.stage_id, st.slot_index, t.school_name, t.team_tag, t.is_seeded, t.created_at
		FROM stage_teams st
		JOIN teams t ON t.id = st.team_id
		WHERE st.stage_id = ?
		ORDER BY
			CASE WHEN st.slot_index > 0 THEN st.slot_index ELSE 999 END ASC,
			t.is_seeded DESC,
			t.id ASC`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		var stgID sql.NullInt64
		var isSeededInt int
		if err := rows.Scan(&t.ID, &t.CategoryID, &stgID, &t.SlotIndex, &t.SchoolName, &t.TeamTag, &isSeededInt, &t.CreatedAt); err != nil {
			return nil, err
		}
		stgCopy := stageID
		t.StageID = &stgCopy
		t.IsSeeded = isSeededInt == 1
		t.DisplayName = models.FormatTeamDisplayName(t.SchoolName, t.TeamTag)
		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fallback if stage_teams was empty for legacy stage_id rows
	if len(teams) == 0 {
		fallbackRows, err := s.db.Conn.Query(`
			SELECT id, category_id, stage_id, group_slot_index, school_name, team_tag, is_seeded, created_at
			FROM teams
			WHERE stage_id = ?
			ORDER BY
				CASE WHEN group_slot_index > 0 THEN group_slot_index ELSE 999 END ASC,
				is_seeded DESC,
				id ASC`, stageID)
		if err != nil {
			return nil, err
		}
		defer fallbackRows.Close()
		for fallbackRows.Next() {
			var t models.Team
			var stgID sql.NullInt64
			var isSeededInt int
			if err := fallbackRows.Scan(&t.ID, &t.CategoryID, &stgID, &t.SlotIndex, &t.SchoolName, &t.TeamTag, &isSeededInt, &t.CreatedAt); err != nil {
				return nil, err
			}
			if stgID.Valid {
				v := stgID.Int64
				t.StageID = &v
			}
			t.IsSeeded = isSeededInt == 1
			t.DisplayName = models.FormatTeamDisplayName(t.SchoolName, t.TeamTag)
			teams = append(teams, t)
		}
	}

	for i := range teams {
		if teams[i].SlotIndex <= 0 {
			teams[i].SlotIndex = i + 1
		}
	}

	if teams == nil {
		teams = []models.Team{}
	}
	return teams, nil
}

func (s *TournamentService) listPotAssignmentsByCategory(categoryID int64) ([]models.PotAssignment, error) {
	rows, err := s.db.Conn.Query(`
		SELECT pa.id, pa.category_id, pa.round_number, pa.stage_id, COALESCE(st.name, ''),
		       pa.team_id, t.school_name, t.team_tag, pa.pot_number, pa.created_at
		FROM pot_assignments pa
		JOIN teams t ON t.id = pa.team_id
		LEFT JOIN stages st ON st.id = pa.stage_id
		WHERE pa.category_id = ?
		ORDER BY pa.pot_number ASC, pa.id ASC`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pots []models.PotAssignment
	for rows.Next() {
		var p models.PotAssignment
		var stgID sql.NullInt64
		if err := rows.Scan(
			&p.ID, &p.CategoryID, &p.RoundNumber, &stgID, &p.StageName,
			&p.TeamID, &p.SchoolName, &p.TeamTag, &p.PotNumber, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		if stgID.Valid {
			v := stgID.Int64
			p.StageID = &v
		}
		p.TeamName = models.FormatTeamDisplayName(p.SchoolName, p.TeamTag)
		pots = append(pots, p)
	}
	if pots == nil {
		pots = []models.PotAssignment{}
	}
	return pots, rows.Err()
}

func (s *TournamentService) listMatchesByStage(stageID int64) ([]models.Match, error) {
	rows, err := s.db.Conn.Query(`
		SELECT m.id, m.category_id, m.stage_id, COALESCE(st.name, ''), m.round, m.match_number,
		       m.bracket_slot, m.source_home_stage_id, m.source_home_rank, m.source_away_stage_id, m.source_away_rank,
		       m.home_team_id, m.away_team_id, m.home_team_name, m.away_team_name,
		       m.is_bye, m.is_forfeit, m.forfeit_winner_team_id, m.home_score, m.away_score,
		       m.home_penalties, m.away_penalties, m.next_match_id, m.next_match_slot, m.winner_team_id,
		       m.status, m.venue, m.court, m.scheduled_at, m.notes, m.updated_at
		FROM matches m
		LEFT JOIN stages st ON st.id = m.stage_id
		WHERE m.stage_id = ?
		ORDER BY m.round ASC, m.match_number ASC, m.id ASC`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatchesRows(rows)
}

func (s *TournamentService) listMatchesByCategory(categoryID int64) ([]models.Match, error) {
	rows, err := s.db.Conn.Query(`
		SELECT m.id, m.category_id, m.stage_id, COALESCE(st.name, ''), m.round, m.match_number,
		       m.bracket_slot, m.source_home_stage_id, m.source_home_rank, m.source_away_stage_id, m.source_away_rank,
		       m.home_team_id, m.away_team_id, m.home_team_name, m.away_team_name,
		       m.is_bye, m.is_forfeit, m.forfeit_winner_team_id, m.home_score, m.away_score,
		       m.home_penalties, m.away_penalties, m.next_match_id, m.next_match_slot, m.winner_team_id,
		       m.status, m.venue, m.court, m.scheduled_at, m.notes, m.updated_at
		FROM matches m
		LEFT JOIN stages st ON st.id = m.stage_id
		WHERE m.category_id = ?
		ORDER BY m.stage_id ASC, m.round ASC, m.match_number ASC, m.id ASC`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatchesRows(rows)
}

func scanMatchesRows(rows *sql.Rows) ([]models.Match, error) {
	var matches []models.Match
	for rows.Next() {
		var m models.Match
		var isByeInt, isForfeitInt int
		var statusStr string
		var homeID, awayID, srcHomeStg, srcAwayStg, forfeitWinID, homePen, awayPen, nextMatchID, winnerTeamID sql.NullInt64

		if err := rows.Scan(
			&m.ID, &m.CategoryID, &m.StageID, &m.StageName, &m.Round, &m.MatchNumber,
			&m.BracketSlot, &srcHomeStg, &m.SourceHomeRank, &srcAwayStg, &m.SourceAwayRank,
			&homeID, &awayID, &m.HomeTeamName, &m.AwayTeamName,
			&isByeInt, &isForfeitInt, &forfeitWinID, &m.HomeScore, &m.AwayScore,
			&homePen, &awayPen, &nextMatchID, &m.NextMatchSlot, &winnerTeamID,
			&statusStr, &m.Venue, &m.Court, &m.ScheduledAt, &m.Notes, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if homeID.Valid {
			v := homeID.Int64
			m.HomeTeamID = &v
		}
		if awayID.Valid {
			v := awayID.Int64
			m.AwayTeamID = &v
		}
		if srcHomeStg.Valid {
			v := srcHomeStg.Int64
			m.SourceHomeStageID = &v
		}
		if srcAwayStg.Valid {
			v := srcAwayStg.Int64
			m.SourceAwayStageID = &v
		}
		if forfeitWinID.Valid {
			v := forfeitWinID.Int64
			m.ForfeitWinnerTeamID = &v
		}
		if homePen.Valid {
			v := int(homePen.Int64)
			m.HomePenalties = &v
		}
		if awayPen.Valid {
			v := int(awayPen.Int64)
			m.AwayPenalties = &v
		}
		if nextMatchID.Valid {
			v := nextMatchID.Int64
			m.NextMatchID = &v
		}
		if winnerTeamID.Valid {
			v := winnerTeamID.Int64
			m.WinnerTeamID = &v
		}
		m.IsBye = isByeInt == 1
		m.IsForfeit = isForfeitInt == 1
		m.Status = models.MatchStatus(statusStr)
		matches = append(matches, m)
	}
	if matches == nil {
		matches = []models.Match{}
	}
	return matches, rows.Err()
}

package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"tourna-desk/internal/engine"
	"tourna-desk/internal/models"
)

type DB struct {
	Conn *sql.DB
}

func Open(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Max open connections set to 1 to prevent SQLite database lock contention
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA foreign_keys = ON;",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to execute %s: %w", p, err)
		}
	}

	db := &DB{Conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	if err := db.seedIfEmpty(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("seed failed: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	if db.Conn != nil {
		return db.Conn.Close()
	}
	return nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tournaments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		location TEXT NOT NULL DEFAULT '',
		start_date TEXT NOT NULL DEFAULT '',
		end_date TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tournament_id INTEGER NOT NULL,
		sport_slug TEXT NOT NULL,
		name TEXT NOT NULL,
		format TEXT NOT NULL DEFAULT 'groups',
		status TEXT NOT NULL DEFAULT 'AWAITING_DRAW',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS stages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		stage_type TEXT NOT NULL DEFAULT 'group',
		status TEXT NOT NULL DEFAULT 'AWAITING_DRAW',
		round_number INTEGER NOT NULL DEFAULT 1,
		parent_stage_id INTEGER,
		double_round_robin INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
		FOREIGN KEY (parent_stage_id) REFERENCES stages(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS teams (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL,
		stage_id INTEGER,
		group_slot_index INTEGER NOT NULL DEFAULT 0,
		school_name TEXT NOT NULL,
		team_tag TEXT NOT NULL DEFAULT '',
		is_seeded INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
		FOREIGN KEY (stage_id) REFERENCES stages(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS stage_teams (
		stage_id INTEGER NOT NULL,
		team_id INTEGER NOT NULL,
		slot_index INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (stage_id, team_id),
		FOREIGN KEY (stage_id) REFERENCES stages(id) ON DELETE CASCADE,
		FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS pot_assignments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL,
		round_number INTEGER NOT NULL DEFAULT 1,
		stage_id INTEGER,
		team_id INTEGER NOT NULL,
		pot_number INTEGER NOT NULL CHECK (pot_number IN (1, 2)),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(category_id, round_number, team_id),
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
		FOREIGN KEY (stage_id) REFERENCES stages(id) ON DELETE SET NULL,
		FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS matches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL,
		stage_id INTEGER NOT NULL,
		round INTEGER NOT NULL,
		match_number INTEGER NOT NULL,
		bracket_slot TEXT NOT NULL DEFAULT '',
		source_home_stage_id INTEGER,
		source_home_rank INTEGER NOT NULL DEFAULT 0,
		source_away_stage_id INTEGER,
		source_away_rank INTEGER NOT NULL DEFAULT 0,
		home_team_id INTEGER,
		away_team_id INTEGER,
		home_team_name TEXT NOT NULL,
		away_team_name TEXT NOT NULL,
		is_bye INTEGER NOT NULL DEFAULT 0,
		is_forfeit INTEGER NOT NULL DEFAULT 0,
		forfeit_winner_team_id INTEGER,
		home_score INTEGER NOT NULL DEFAULT 0,
		away_score INTEGER NOT NULL DEFAULT 0,
		home_penalties INTEGER,
		away_penalties INTEGER,
		next_match_id INTEGER,
		next_match_slot TEXT NOT NULL DEFAULT '',
		winner_team_id INTEGER,
		status TEXT NOT NULL DEFAULT 'scheduled',
		venue TEXT NOT NULL DEFAULT '',
		court TEXT NOT NULL DEFAULT '',
		scheduled_at TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
		FOREIGN KEY (stage_id) REFERENCES stages(id) ON DELETE CASCADE,
		FOREIGN KEY (home_team_id) REFERENCES teams(id) ON DELETE SET NULL,
		FOREIGN KEY (away_team_id) REFERENCES teams(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS venues (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		address TEXT,
		supported_sports TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_categories_tournament ON categories(tournament_id);
	CREATE INDEX IF NOT EXISTS idx_stages_category ON stages(category_id);
	CREATE INDEX IF NOT EXISTS idx_teams_category ON teams(category_id);
	CREATE INDEX IF NOT EXISTS idx_teams_stage ON teams(stage_id);
	CREATE INDEX IF NOT EXISTS idx_stage_teams_stage ON stage_teams(stage_id);
	CREATE INDEX IF NOT EXISTS idx_stage_teams_team ON stage_teams(team_id);
	CREATE INDEX IF NOT EXISTS idx_pot_assignments_category ON pot_assignments(category_id, round_number);
	CREATE INDEX IF NOT EXISTS idx_matches_stage ON matches(stage_id);
	CREATE INDEX IF NOT EXISTS idx_matches_category ON matches(category_id);
	`
	if _, err := db.Conn.Exec(schema); err != nil {
		return err
	}

	// Ensure backward-compatible migration if an older teams table was created with (name, school, coach)
	var hasSchoolName int
	_ = db.Conn.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('teams') WHERE name = 'school_name'`).Scan(&hasSchoolName)
	if hasSchoolName == 0 {
		_, err := db.Conn.Exec(`
			ALTER TABLE teams RENAME TO teams_old;
			CREATE TABLE teams (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				category_id INTEGER NOT NULL,
				stage_id INTEGER,
				school_name TEXT NOT NULL,
				team_tag TEXT NOT NULL DEFAULT '',
				is_seeded INTEGER NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
				FOREIGN KEY (stage_id) REFERENCES stages(id) ON DELETE SET NULL
			);
			INSERT INTO teams (id, category_id, stage_id, school_name, team_tag, created_at)
			SELECT id, category_id, stage_id, name, '', created_at FROM teams_old;
			DROP TABLE teams_old;
		`)
		if err != nil {
			return err
		}
	}

	// Incremental column migrations for existing databases
	alterColumns := []struct {
		table  string
		column string
		ddl    string
	}{
		{"categories", "format", "ALTER TABLE categories ADD COLUMN format TEXT NOT NULL DEFAULT 'groups'"},
		{"categories", "status", "ALTER TABLE categories ADD COLUMN status TEXT NOT NULL DEFAULT 'AWAITING_DRAW'"},
		{"stages", "status", "ALTER TABLE stages ADD COLUMN status TEXT NOT NULL DEFAULT 'AWAITING_DRAW'"},
		{"stages", "round_number", "ALTER TABLE stages ADD COLUMN round_number INTEGER NOT NULL DEFAULT 1"},
		{"stages", "parent_stage_id", "ALTER TABLE stages ADD COLUMN parent_stage_id INTEGER"},
		{"teams", "is_seeded", "ALTER TABLE teams ADD COLUMN is_seeded INTEGER NOT NULL DEFAULT 0"},
		{"teams", "group_slot_index", "ALTER TABLE teams ADD COLUMN group_slot_index INTEGER NOT NULL DEFAULT 0"},
		{"stage_teams", "slot_index", "ALTER TABLE stage_teams ADD COLUMN slot_index INTEGER NOT NULL DEFAULT 0"},
		{"matches", "bracket_slot", "ALTER TABLE matches ADD COLUMN bracket_slot TEXT NOT NULL DEFAULT ''"},
		{"matches", "source_home_stage_id", "ALTER TABLE matches ADD COLUMN source_home_stage_id INTEGER"},
		{"matches", "source_home_rank", "ALTER TABLE matches ADD COLUMN source_home_rank INTEGER NOT NULL DEFAULT 0"},
		{"matches", "source_away_stage_id", "ALTER TABLE matches ADD COLUMN source_away_stage_id INTEGER"},
		{"matches", "source_away_rank", "ALTER TABLE matches ADD COLUMN source_away_rank INTEGER NOT NULL DEFAULT 0"},
		{"matches", "is_forfeit", "ALTER TABLE matches ADD COLUMN is_forfeit INTEGER NOT NULL DEFAULT 0"},
		{"matches", "forfeit_winner_team_id", "ALTER TABLE matches ADD COLUMN forfeit_winner_team_id INTEGER"},
		{"matches", "home_penalties", "ALTER TABLE matches ADD COLUMN home_penalties INTEGER"},
		{"matches", "away_penalties", "ALTER TABLE matches ADD COLUMN away_penalties INTEGER"},
		{"matches", "next_match_id", "ALTER TABLE matches ADD COLUMN next_match_id INTEGER"},
		{"matches", "next_match_slot", "ALTER TABLE matches ADD COLUMN next_match_slot TEXT NOT NULL DEFAULT ''"},
		{"matches", "winner_team_id", "ALTER TABLE matches ADD COLUMN winner_team_id INTEGER"},
	}
	for _, ac := range alterColumns {
		var colExists int
		_ = db.Conn.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?`, ac.table), ac.column).Scan(&colExists)
		if colExists == 0 {
			if _, err := db.Conn.Exec(ac.ddl); err != nil {
				return err
			}
		}
	}

	// Backfill stage_teams from teams.stage_id for any existing assignments
	if _, err := db.Conn.Exec(`
		INSERT OR IGNORE INTO stage_teams (stage_id, team_id)
		SELECT stage_id, id FROM teams WHERE stage_id IS NOT NULL
	`); err != nil {
		return err
	}

	return nil
}

func (db *DB) seedIfEmpty() error {
	var venueCount int
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM venues").Scan(&venueCount); err == nil && venueCount == 0 {
		defaultVenues := []struct {
			name    string
			address string
			sports  string
		}{
			{"Atatürk Spor Salonu", "Merkez Spor Kompleksi", `["BASKETBOL","VOLEYBOL","FUTSAL","HENTBOL"]`},
			{"Cumhuriyet Voleybol Salonu", "Cumhuriyet Mah.", `["VOLEYBOL"]`},
			{"Merkez Basketbol Arena", "Stadyum Yanı", `["BASKETBOL"]`},
			{"19 Mayıs Stadyumu & Halı Saha", "Gençlik Caddesi", `["FUTBOL","FUTSAL"]`},
		}
		for _, v := range defaultVenues {
			_, _ = db.Conn.Exec(`INSERT INTO venues (name, address, supported_sports) VALUES (?, ?, ?)`, v.name, v.address, v.sports)
		}
	}

	var count int
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM tournaments").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO tournaments (name, description, location, start_date, end_date)
		VALUES (?, ?, ?, ?, ?)`,
		"2026-2027 Okul Sporları Sezonu",
		"Ortaöğretim ve Yıldızlar Okullararası Spor Sezonu",
		"Atatürk Spor Kompleksi",
		"2026-10-19",
		"2026-10-30",
	)
	if err != nil {
		return err
	}
	tournID, _ := res.LastInsertId()

	// 1. Futsal Category: Genç Erkek A
	catFootRes, err := tx.Exec(`
		INSERT INTO categories (tournament_id, sport_slug, name) VALUES (?, ?, ?)`,
		tournID, string(models.SportFutsal), "Genç Erkek A (Futsal)",
	)
	if err != nil {
		return err
	}
	catFootID, _ := catFootRes.LastInsertId()

	stgFootRes, err := tx.Exec(`
		INSERT INTO stages (category_id, name, stage_type, double_round_robin) VALUES (?, ?, 'group', 0)`,
		catFootID, "Grup A",
	)
	if err != nil {
		return err
	}
	stgFootID, _ := stgFootRes.LastInsertId()

	footSchoolNames := []string{
		"Atatürk Anadolu Lisesi",
		"Cumhuriyet Fen Lisesi",
		"Anadolu İmam Hatip Lisesi",
		"Fatih Mesleki Teknik AL",
	}
	var footTeams []models.Team
	for _, schoolName := range footSchoolNames {
		tr, err := tx.Exec(`INSERT INTO teams (category_id, stage_id, school_name) VALUES (?, ?, ?)`,
			catFootID, stgFootID, schoolName)
		if err != nil {
			return err
		}
		tid, _ := tr.LastInsertId()
		stgIDCopy := stgFootID
		footTeams = append(footTeams, models.Team{
			ID:          tid,
			CategoryID:  catFootID,
			StageID:     &stgIDCopy,
			SchoolName:  schoolName,
			DisplayName: models.FormatTeamDisplayName(schoolName),
		})
	}

	footMatches := engine.GenerateRoundRobin(catFootID, stgFootID, footTeams, false)
	for i := range footMatches {
		footMatches[i].Venue = "Atatürk Spor Salonu"
		footMatches[i].Court = "Ana Saha"
		footMatches[i].ScheduledAt = fmt.Sprintf("2026-10-%d %02d:00", 19+(footMatches[i].Round-1), 10+(i%2)*2)
		if i == 0 {
			footMatches[i].HomeScore = 3
			footMatches[i].AwayScore = 1
			footMatches[i].Status = models.StatusCompleted
		} else if i == 1 {
			footMatches[i].HomeScore = 2
			footMatches[i].AwayScore = 2
			hp, ap := 4, 3
			footMatches[i].HomePenalties = &hp
			footMatches[i].AwayPenalties = &ap
			footMatches[i].Status = models.StatusCompleted
		}
		if err := insertMatchTx(tx, footMatches[i]); err != nil {
			return err
		}
	}

	// 2. Volleyball Category: Yıldız Kız
	catVolRes, err := tx.Exec(`
		INSERT INTO categories (tournament_id, sport_slug, name) VALUES (?, ?, ?)`,
		tournID, string(models.SportVolleyball), "Yıldız Kız (Voleybol)",
	)
	if err != nil {
		return err
	}
	catVolID, _ := catVolRes.LastInsertId()

	stgVolRes, err := tx.Exec(`
		INSERT INTO stages (category_id, name, stage_type, double_round_robin) VALUES (?, ?, 'group', 0)`,
		catVolID, "Grup A",
	)
	if err != nil {
		return err
	}
	stgVolID, _ := stgVolRes.LastInsertId()

	volSchoolNames := []string{
		"Gazi Ortaokulu",
		"Mimar Sinan Ortaokulu",
		"Yunus Emre Ortaokulu",
		"İstiklal Ortaokulu",
	}
	var volTeams []models.Team
	for _, schoolName := range volSchoolNames {
		tr, err := tx.Exec(`INSERT INTO teams (category_id, stage_id, school_name) VALUES (?, ?, ?)`,
			catVolID, stgVolID, schoolName)
		if err != nil {
			return err
		}
		tid, _ := tr.LastInsertId()
		stgIDCopy := stgVolID
		volTeams = append(volTeams, models.Team{
			ID:          tid,
			CategoryID:  catVolID,
			StageID:     &stgIDCopy,
			SchoolName:  schoolName,
			DisplayName: models.FormatTeamDisplayName(schoolName),
		})
	}

	volMatches := engine.GenerateRoundRobin(catVolID, stgVolID, volTeams, false)
	for i := range volMatches {
		volMatches[i].Venue = "Cumhuriyet Voleybol Salonu"
		volMatches[i].Court = "Kort 1"
		volMatches[i].ScheduledAt = fmt.Sprintf("2026-10-%d %02d:30", 20+(volMatches[i].Round-1), 11+(i%2)*2)
		if i == 0 {
			volMatches[i].HomeScore = 3
			volMatches[i].AwayScore = 1
			volMatches[i].Status = models.StatusCompleted
		} else if i == 1 {
			volMatches[i].HomeScore = 3
			volMatches[i].AwayScore = 2
			volMatches[i].Status = models.StatusCompleted
		}
		if err := insertMatchTx(tx, volMatches[i]); err != nil {
			return err
		}
	}

	// 3. Basketball Category: Genç Erkekler Basketbol
	catBaskRes, err := tx.Exec(`
		INSERT INTO categories (tournament_id, sport_slug, name) VALUES (?, ?, ?)`,
		tournID, string(models.SportBasketball), "Genç Erkekler Basketbol",
	)
	if err != nil {
		return err
	}
	catBaskID, _ := catBaskRes.LastInsertId()

	stgBaskRes, err := tx.Exec(`
		INSERT INTO stages (category_id, name, stage_type, double_round_robin) VALUES (?, ?, 'group', 0)`,
		catBaskID, "Grup A",
	)
	if err != nil {
		return err
	}
	stgBaskID, _ := stgBaskRes.LastInsertId()

	baskSchoolNames := []string{
		"TED Koleji",
		"Bahçeşehir Koleji",
		"Fen Bilimleri Lisesi",
		"Merkez Anadolu Lisesi",
	}
	var baskTeams []models.Team
	for _, schoolName := range baskSchoolNames {
		tr, err := tx.Exec(`INSERT INTO teams (category_id, stage_id, school_name) VALUES (?, ?, ?)`,
			catBaskID, stgBaskID, schoolName)
		if err != nil {
			return err
		}
		tid, _ := tr.LastInsertId()
		stgIDCopy := stgBaskID
		baskTeams = append(baskTeams, models.Team{
			ID:          tid,
			CategoryID:  catBaskID,
			StageID:     &stgIDCopy,
			SchoolName:  schoolName,
			DisplayName: models.FormatTeamDisplayName(schoolName),
		})
	}

	baskMatches := engine.GenerateRoundRobin(catBaskID, stgBaskID, baskTeams, false)
	for i := range baskMatches {
		baskMatches[i].Venue = "Merkez Basketbol Arena"
		baskMatches[i].Court = "Parke 1"
		baskMatches[i].ScheduledAt = fmt.Sprintf("2026-10-%d %02d:00", 21+(baskMatches[i].Round-1), 13+(i%2)*2)
		if i == 0 {
			baskMatches[i].HomeScore = 74
			baskMatches[i].AwayScore = 68
			baskMatches[i].Status = models.StatusCompleted
		}
		if err := insertMatchTx(tx, baskMatches[i]); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO stage_teams (stage_id, team_id)
		SELECT stage_id, id FROM teams WHERE stage_id IS NOT NULL
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func insertMatchTx(tx *sql.Tx, m models.Match) error {
	isByeInt := 0
	if m.IsBye {
		isByeInt = 1
	}
	isForfeitInt := 0
	if m.IsForfeit {
		isForfeitInt = 1
	}
	_, err := tx.Exec(`
		INSERT INTO matches (
			category_id, stage_id, round, match_number, bracket_slot,
			source_home_stage_id, source_home_rank, source_away_stage_id, source_away_rank,
			home_team_id, away_team_id, home_team_name, away_team_name,
			is_bye, is_forfeit, forfeit_winner_team_id, home_score, away_score,
			home_penalties, away_penalties, next_match_id, next_match_slot, winner_team_id,
			status, venue, court, scheduled_at, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.CategoryID, m.StageID, m.Round, m.MatchNumber, m.BracketSlot,
		m.SourceHomeStageID, m.SourceHomeRank, m.SourceAwayStageID, m.SourceAwayRank,
		m.HomeTeamID, m.AwayTeamID, m.HomeTeamName, m.AwayTeamName,
		isByeInt, isForfeitInt, m.ForfeitWinnerTeamID, m.HomeScore, m.AwayScore,
		m.HomePenalties, m.AwayPenalties, m.NextMatchID, m.NextMatchSlot, m.WinnerTeamID,
		string(m.Status), m.Venue, m.Court, m.ScheduledAt, m.Notes,
	)
	return err
}

package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	conn.SetMaxOpenConns(1) // SQLite single writer
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON`,
		`CREATE TABLE IF NOT EXISTS rotation (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			user_id  TEXT NOT NULL,
			username TEXT NOT NULL,
			position INTEGER NOT NULL,
			active   INTEGER NOT NULL DEFAULT 1,
			UNIQUE(guild_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS proposed_dates (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id    TEXT NOT NULL,
			date        TEXT NOT NULL,
			proposed_by TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS polls (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id     TEXT NOT NULL,
			message_id   TEXT,
			channel_id   TEXT NOT NULL,
			status       TEXT NOT NULL DEFAULT 'open',
			winning_date TEXT,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS poll_options (
			id      INTEGER PRIMARY KEY AUTOINCREMENT,
			poll_id INTEGER NOT NULL REFERENCES polls(id),
			emoji   TEXT NOT NULL,
			date    TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS brews (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id         TEXT NOT NULL,
			name             TEXT,
			brewer_id        TEXT NOT NULL,
			brewer_name      TEXT NOT NULL,
			date             TEXT,
			channel_id       TEXT,
			stats_message_id TEXT,
			status           TEXT NOT NULL DEFAULT 'scheduled',
			created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Migration: add stats_message_id to existing DBs
		`ALTER TABLE brews ADD COLUMN stats_message_id TEXT`,
		`CREATE TABLE IF NOT EXISTS recipes (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			brew_id     INTEGER NOT NULL UNIQUE REFERENCES brews(id),
			style       TEXT,
			og          REAL,
			fg          REAL,
			abv         REAL,
			ingredients TEXT,
			notes       TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ratings (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			brew_id    INTEGER NOT NULL REFERENCES brews(id),
			user_id    TEXT NOT NULL,
			username   TEXT NOT NULL,
			rating     INTEGER NOT NULL CHECK(rating >= 1 AND rating <= 5),
			notes      TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(brew_id, user_id)
		)`,
	}
	for _, s := range stmts {
		if _, err := d.conn.Exec(s); err != nil {
			// ALTER TABLE fails if column already exists — safe to ignore
			if s[:min(5, len(s))] == "ALTER" {
				continue
			}
			return fmt.Errorf("stmt %q: %w", s[:min(40, len(s))], err)
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- Types ---

type RotationMember struct {
	ID       int64
	GuildID  string
	UserID   string
	Username string
	Position int
	Active   bool
}

type ProposedDate struct {
	ID         int64
	GuildID    string
	Date       string
	ProposedBy string
}

type Poll struct {
	ID          int64
	GuildID     string
	MessageID   string
	ChannelID   string
	Status      string
	WinningDate string
}

type PollOption struct {
	ID     int64
	PollID int64
	Emoji  string
	Date   string
}

type Brew struct {
	ID             int64
	GuildID        string
	Name           string
	BrewerID       string
	BrewerName     string
	Date           string
	ChannelID      string
	StatsMessageID string
	Status         string
}

type Recipe struct {
	ID          int64
	BrewID      int64
	Style       string
	OG          float64
	FG          float64
	ABV         float64
	Ingredients string
	Notes       string
}

type Rating struct {
	ID       int64
	BrewID   int64
	UserID   string
	Username string
	Rating   int
	Notes    string
}

type BlackboardEntry struct {
	Brew      Brew
	Recipe    *Recipe
	AvgRating float64
	Ratings   []Rating
}

// --- Rotation ---

func (d *DB) GetRotation(guildID string) ([]RotationMember, error) {
	rows, err := d.conn.Query(
		`SELECT id,guild_id,user_id,username,position,active FROM rotation WHERE guild_id=? ORDER BY position`,
		guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []RotationMember
	for rows.Next() {
		var m RotationMember
		var active int
		if err := rows.Scan(&m.ID, &m.GuildID, &m.UserID, &m.Username, &m.Position, &active); err != nil {
			return nil, err
		}
		m.Active = active == 1
		members = append(members, m)
	}
	return members, rows.Err()
}

func (d *DB) AddRotationMember(guildID, userID, username string) error {
	var maxPos int
	d.conn.QueryRow(`SELECT COALESCE(MAX(position),0) FROM rotation WHERE guild_id=?`, guildID).Scan(&maxPos)
	_, err := d.conn.Exec(
		`INSERT INTO rotation(guild_id,user_id,username,position) VALUES(?,?,?,?)
		 ON CONFLICT(guild_id,user_id) DO UPDATE SET username=excluded.username, active=1`,
		guildID, userID, username, maxPos+1,
	)
	return err
}

func (d *DB) NextBrewer(guildID string) (*RotationMember, error) {
	var m RotationMember
	var active int
	err := d.conn.QueryRow(
		`SELECT id,guild_id,user_id,username,position,active FROM rotation WHERE guild_id=? AND active=1 ORDER BY position LIMIT 1`,
		guildID,
	).Scan(&m.ID, &m.GuildID, &m.UserID, &m.Username, &m.Position, &active)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Active = active == 1
	return &m, nil
}

func (d *DB) SkipBrewer(guildID, userID string) error {
	var maxPos int
	d.conn.QueryRow(`SELECT COALESCE(MAX(position),0) FROM rotation WHERE guild_id=?`, guildID).Scan(&maxPos)
	_, err := d.conn.Exec(
		`UPDATE rotation SET position=? WHERE guild_id=? AND user_id=?`,
		maxPos+1, guildID, userID,
	)
	return err
}

// --- Proposed Dates ---

func (d *DB) AddProposedDates(guildID, proposedBy string, dates []string) error {
	for _, date := range dates {
		if _, err := d.conn.Exec(
			`INSERT INTO proposed_dates(guild_id,date,proposed_by) VALUES(?,?,?)`,
			guildID, date, proposedBy,
		); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) GetProposedDates(guildID string) ([]ProposedDate, error) {
	rows, err := d.conn.Query(
		`SELECT id,guild_id,date,proposed_by FROM proposed_dates WHERE guild_id=? ORDER BY created_at`,
		guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dates []ProposedDate
	for rows.Next() {
		var pd ProposedDate
		if err := rows.Scan(&pd.ID, &pd.GuildID, &pd.Date, &pd.ProposedBy); err != nil {
			return nil, err
		}
		dates = append(dates, pd)
	}
	return dates, rows.Err()
}

func (d *DB) ClearProposedDates(guildID string) error {
	_, err := d.conn.Exec(`DELETE FROM proposed_dates WHERE guild_id=?`, guildID)
	return err
}

// --- Polls ---

func (d *DB) CreatePoll(guildID, channelID string) (int64, error) {
	res, err := d.conn.Exec(
		`INSERT INTO polls(guild_id,channel_id) VALUES(?,?)`,
		guildID, channelID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) SetPollMessage(pollID int64, messageID string) error {
	_, err := d.conn.Exec(`UPDATE polls SET message_id=? WHERE id=?`, messageID, pollID)
	return err
}

func (d *DB) AddPollOption(pollID int64, emoji, date string) error {
	_, err := d.conn.Exec(
		`INSERT INTO poll_options(poll_id,emoji,date) VALUES(?,?,?)`,
		pollID, emoji, date,
	)
	return err
}

func (d *DB) GetOpenPoll(guildID string) (*Poll, error) {
	var p Poll
	err := d.conn.QueryRow(
		`SELECT id,guild_id,COALESCE(message_id,''),channel_id,status,COALESCE(winning_date,'')
		 FROM polls WHERE guild_id=? AND status='open' ORDER BY id DESC LIMIT 1`,
		guildID,
	).Scan(&p.ID, &p.GuildID, &p.MessageID, &p.ChannelID, &p.Status, &p.WinningDate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (d *DB) GetPollOptions(pollID int64) ([]PollOption, error) {
	rows, err := d.conn.Query(
		`SELECT id,poll_id,emoji,date FROM poll_options WHERE poll_id=?`,
		pollID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var opts []PollOption
	for rows.Next() {
		var o PollOption
		rows.Scan(&o.ID, &o.PollID, &o.Emoji, &o.Date)
		opts = append(opts, o)
	}
	return opts, rows.Err()
}

func (d *DB) ClosePoll(pollID int64, winningDate string) error {
	_, err := d.conn.Exec(
		`UPDATE polls SET status='closed',winning_date=? WHERE id=?`,
		winningDate, pollID,
	)
	return err
}

func (d *DB) GetPollByMessage(messageID string) (*Poll, []PollOption, error) {
	var p Poll
	err := d.conn.QueryRow(
		`SELECT id,guild_id,COALESCE(message_id,''),channel_id,status,COALESCE(winning_date,'')
		 FROM polls WHERE message_id=?`,
		messageID,
	).Scan(&p.ID, &p.GuildID, &p.MessageID, &p.ChannelID, &p.Status, &p.WinningDate)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	opts, err := d.GetPollOptions(p.ID)
	return &p, opts, err
}

// --- Brews ---

func (d *DB) CreateBrew(guildID, brewerID, brewerName, date string) (int64, error) {
	res, err := d.conn.Exec(
		`INSERT INTO brews(guild_id,brewer_id,brewer_name,date) VALUES(?,?,?,?)`,
		guildID, brewerID, brewerName, date,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) SetBrewChannel(brewID int64, channelID string) error {
	_, err := d.conn.Exec(
		`UPDATE brews SET channel_id=?,status='active' WHERE id=?`,
		channelID, brewID,
	)
	return err
}

func (d *DB) SetBrewName(brewID int64, name string) error {
	_, err := d.conn.Exec(`UPDATE brews SET name=? WHERE id=?`, name, brewID)
	return err
}

func (d *DB) GetBrewByChannel(channelID string) (*Brew, error) {
	var b Brew
	err := d.conn.QueryRow(
		`SELECT id,guild_id,COALESCE(name,''),brewer_id,brewer_name,COALESCE(date,''),COALESCE(channel_id,''),COALESCE(stats_message_id,''),status
		 FROM brews WHERE channel_id=?`,
		channelID,
	).Scan(&b.ID, &b.GuildID, &b.Name, &b.BrewerID, &b.BrewerName, &b.Date, &b.ChannelID, &b.StatsMessageID, &b.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

func (d *DB) SetBrewStatsMessage(brewID int64, messageID string) error {
	_, err := d.conn.Exec(`UPDATE brews SET stats_message_id=? WHERE id=?`, messageID, brewID)
	return err
}

func (d *DB) CompleteBrew(brewID int64) error {
	_, err := d.conn.Exec(`UPDATE brews SET status='complete' WHERE id=?`, brewID)
	return err
}

// --- Recipes ---

func (d *DB) UpsertRecipe(brewID int64, style string, og, fg, abv float64, ingredients, notes string) error {
	d.conn.Exec(`DELETE FROM recipes WHERE brew_id=?`, brewID)
	_, err := d.conn.Exec(
		`INSERT INTO recipes(brew_id,style,og,fg,abv,ingredients,notes) VALUES(?,?,?,?,?,?,?)`,
		brewID, style, og, fg, abv, ingredients, notes,
	)
	return err
}

// SetFinalGravity updates FG and recalculates ABV on an existing recipe.
func (d *DB) SetFinalGravity(brewID int64, fg float64) (float64, error) {
	var og float64
	err := d.conn.QueryRow(`SELECT COALESCE(og,0) FROM recipes WHERE brew_id=?`, brewID).Scan(&og)
	if err != nil {
		return 0, err
	}
	abv := 0.0
	if og > fg {
		abv = (og - fg) * 131.25
	}
	_, err = d.conn.Exec(`UPDATE recipes SET fg=?,abv=? WHERE brew_id=?`, fg, abv, brewID)
	return abv, err
}

func (d *DB) GetRecipe(brewID int64) (*Recipe, error) {
	var r Recipe
	err := d.conn.QueryRow(
		`SELECT id,brew_id,COALESCE(style,''),COALESCE(og,0),COALESCE(fg,0),COALESCE(abv,0),COALESCE(ingredients,''),COALESCE(notes,'')
		 FROM recipes WHERE brew_id=?`,
		brewID,
	).Scan(&r.ID, &r.BrewID, &r.Style, &r.OG, &r.FG, &r.ABV, &r.Ingredients, &r.Notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// --- Ratings ---

func (d *DB) UpsertRating(brewID int64, userID, username string, rating int, notes string) error {
	_, err := d.conn.Exec(
		`INSERT INTO ratings(brew_id,user_id,username,rating,notes) VALUES(?,?,?,?,?)
		 ON CONFLICT(brew_id,user_id) DO UPDATE SET rating=excluded.rating,notes=excluded.notes,username=excluded.username`,
		brewID, userID, username, rating, notes,
	)
	return err
}

func (d *DB) GetRatings(brewID int64) ([]Rating, error) {
	rows, err := d.conn.Query(
		`SELECT id,brew_id,user_id,username,rating,COALESCE(notes,'') FROM ratings WHERE brew_id=? ORDER BY created_at`,
		brewID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ratings []Rating
	for rows.Next() {
		var r Rating
		rows.Scan(&r.ID, &r.BrewID, &r.UserID, &r.Username, &r.Rating, &r.Notes)
		ratings = append(ratings, r)
	}
	return ratings, rows.Err()
}

// --- Blackboard ---

func (d *DB) GetBlackboard(guildID string) ([]BlackboardEntry, error) {
	rows, err := d.conn.Query(`
		SELECT b.id,b.guild_id,COALESCE(b.name,'Unnamed'),b.brewer_id,b.brewer_name,
		       COALESCE(b.date,''),COALESCE(b.channel_id,''),b.status,
		       COALESCE(AVG(r.rating),0)
		FROM brews b
		LEFT JOIN ratings r ON r.brew_id=b.id
		WHERE b.guild_id=? AND b.status='complete'
		GROUP BY b.id
		ORDER BY b.created_at DESC`,
		guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []BlackboardEntry
	for rows.Next() {
		var e BlackboardEntry
		rows.Scan(
			&e.Brew.ID, &e.Brew.GuildID, &e.Brew.Name, &e.Brew.BrewerID, &e.Brew.BrewerName,
			&e.Brew.Date, &e.Brew.ChannelID, &e.Brew.Status, &e.AvgRating,
		)
		e.Recipe, _ = d.GetRecipe(e.Brew.ID)
		e.Ratings, _ = d.GetRatings(e.Brew.ID)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

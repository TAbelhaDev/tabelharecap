package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Item is one "novidade" registered by some external source — a job, a
// taglue workflow step, a one-off script. tarecap never interprets Link; it
// only stores and displays whatever the caller sent.
type Item struct {
	ID        int64
	Source    string
	Title     string
	Body      string
	Link      string
	CreatedAt time.Time
	SeenAt    *time.Time
}

// Store persists items in a local SQLite database.
type Store struct {
	db *sql.DB
}

func openStore() (*Store, error) {
	return openStoreAt(defaultDBPath())
}

// openStoreAt opens (creating if needed) the SQLite database at path in WAL
// mode with a busy timeout. Unlike a single long-lived TUI process, tarecap's
// database is written by many independent short-lived processes (cron jobs,
// taglue workflow steps, ad-hoc scripts) at arbitrary times — WAL lets a
// reader (the TUI browsing the feed) proceed without blocking on a
// concurrent writer, and busy_timeout absorbs the rare writer-vs-writer
// collision instead of surfacing SQLITE_BUSY to a caller's script.
func openStoreAt(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("criar %s: %w", filepath.Dir(path), err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS items (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	source     TEXT NOT NULL,
	title      TEXT NOT NULL,
	body       TEXT,
	link       TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	seen_at    DATETIME
);
CREATE INDEX IF NOT EXISTS idx_items_seen ON items(seen_at);
CREATE INDEX IF NOT EXISTS idx_items_created ON items(created_at);`)
	return err
}

func (s *Store) close() { s.db.Close() }

// add inserts a new item and returns it with its assigned ID and timestamp.
func (s *Store) add(source, title, body, link string) (Item, error) {
	res, err := s.db.Exec(`INSERT INTO items (source, title, body, link) VALUES (?, ?, ?, ?)`,
		source, title, nullIfEmpty(body), nullIfEmpty(link))
	if err != nil {
		return Item{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Item{}, err
	}
	return s.get(id)
}

func (s *Store) get(id int64) (Item, error) {
	row := s.db.QueryRow(`SELECT id, source, title, body, link, created_at, seen_at FROM items WHERE id = ?`, id)
	return scanItem(row)
}

// list returns items newest-first, optionally filtered to unseen-only and/or
// a specific source, capped at limit (0 means no cap).
func (s *Store) list(unseenOnly bool, source string, limit int) ([]Item, error) {
	q := `SELECT id, source, title, body, link, created_at, seen_at FROM items WHERE 1=1`
	var args []any
	if unseenOnly {
		q += ` AND seen_at IS NULL`
	}
	if source != "" {
		q += ` AND source = ?`
		args = append(args, source)
	}
	q += ` ORDER BY created_at DESC, id DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// markSeen sets seen_at = now() for one item. Idempotent: marking an
// already-seen item again is not an error and does not update its timestamp.
func (s *Store) markSeen(id int64) (Item, error) {
	_, err := s.db.Exec(`UPDATE items SET seen_at = CURRENT_TIMESTAMP WHERE id = ? AND seen_at IS NULL`, id)
	if err != nil {
		return Item{}, err
	}
	return s.get(id)
}

// markAllSeen marks every currently-unseen item as seen and reports how many
// rows changed.
func (s *Store) markAllSeen() (int, error) {
	res, err := s.db.Exec(`UPDATE items SET seen_at = CURRENT_TIMESTAMP WHERE seen_at IS NULL`)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// sources returns the distinct source values present, alphabetically —
// tarecap ipc item.add's source= filter is free-text, so this is the only
// way the TUI can offer a "cycle through sources" filter.
func (s *Store) sources() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT source FROM items ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var src string
		if err := rows.Scan(&src); err != nil {
			return nil, err
		}
		out = append(out, src)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanItem(row scannable) (Item, error) {
	var item Item
	var body, link sql.NullString
	var seenAt sql.NullTime
	if err := row.Scan(&item.ID, &item.Source, &item.Title, &body, &link, &item.CreatedAt, &seenAt); err != nil {
		return Item{}, err
	}
	item.Body = body.String
	item.Link = link.String
	if seenAt.Valid {
		item.SeenAt = &seenAt.Time
	}
	return item, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

package store

import "fmt"

const (
	LobbyName                   = "lobby"
	DefaultPublicRetentionDays  = 60
	DefaultPrivateRetentionDays = 45
	DMRetentionDays = 30
	DefaultPrivateCapBytes = 10 << 20
)

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		nick              TEXT NOT NULL UNIQUE COLLATE NOCASE,
		banned            INTEGER NOT NULL DEFAULT 0,
		max_private_rooms INTEGER NOT NULL DEFAULT 3,
		nick_changed_at INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS user_keys (
		fingerprint    TEXT PRIMARY KEY,
		user_id        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		authorized_key TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE COLLATE NOCASE,
		private INTEGER NOT NULL DEFAULT 0,
		owner_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		pin_hash TEXT NOT NULL DEFAULT '',
		cap_bytes INTEGER NOT NULL DEFAULT 0,
		retention_days INTEGER NOT NULL,
		created_at     INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS room_members (
		room_id INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY (room_id, user_id)
	)`,
	`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
		user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		nick TEXT NOT NULL,
		body TEXT NOT NULL,
		size INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_messages_room ON messages (room_id, id)`,
	`CREATE TABLE IF NOT EXISTS dms (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		from_nick  TEXT NOT NULL,
		to_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		body       TEXT NOT NULL,
		created_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS pin_attempts (
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		room_name TEXT NOT NULL COLLATE NOCASE,
		fails INTEGER NOT NULL DEFAULT 0,
		locked_until INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (user_id, room_name)
	)`,
}

func (s *Store) migrate() error {
	for _, stmt := range schemaStatements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO rooms (name, private, retention_days, created_at) VALUES (?, 0, ?, ?)`,
		LobbyName, DefaultPublicRetentionDays, s.Now().Unix())
	if err != nil { return fmt.Errorf("seed lobby: %w", err) }
	return nil
}

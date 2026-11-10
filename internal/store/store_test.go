package store

import (
	"path/filepath"
	"testing"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}
func TestOpenCreatesTables(t *testing.T) {
	st := openTest(t)
	for _, table := range []string{
		"users", "user_keys", "rooms", "room_members", "messages", "dms", "pin_attempts",
	} {
		var n int
		err := st.db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table %s missing", table)
		}
	}
}

func TestOpenSeedsLobby(t *testing.T) {
	st := openTest(t)
	var name string
	var private, retention int
	err := st.db.QueryRow(
		`SELECT name, private, retention_days FROM rooms WHERE name = ?`, LobbyName).
		Scan(&name, &private, &retention)
	if err != nil {
		t.Fatalf("lobby missing: %v", err)
	}
	if private != 0 || retention != DefaultPublicRetentionDays {
		t.Errorf("lobby private=%d retention=%d", private, retention)
	}
}

func TestReopenKeepsOneLobby(t *testing.T) {
	path := filepath.Join(t.TempDir(), "again.db")
	for i := 0; i < 2; i++ {
		st, err := Open(path)
		if err != nil {
			t.Fatalf("Open #%d: %v", i, err)
		}
		st.Close()
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var n int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM rooms`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("rooms = %d, want 1", n)
	}
}


func TestForeignKeysAreOn(t *testing.T) {
	st := openTest(t)
	var on int
	if err := st.db.QueryRow(`PRAGMA foreign_keys`).Scan(&on); err != nil {
		t.Fatal(err)
	}
	if on != 1 {
		t.Errorf("foreign_keys = %d, want 1", on)
	}
}

package store

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	roomRe = regexp.MustCompile(`^[a-z0-9-]{2,24}$`)
	pinRe  = regexp.MustCompile(`^[0-9]{6}$`)
)

func NormalizeRoomName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.TrimPrefix(s, "#")
}

func ValidRoomName(n string) bool {
	return roomRe.MatchString(n)
}

func ValidPIN(p string) bool {
	return pinRe.MatchString(p)
}

const roomCols = `id, name, private, owner_id, pin_hash, cap_bytes, retention_days, created_at`

func scanRoom(r scanner) (Room, error) {
	var rm Room
	var private int
	var owner sql.NullInt64
	var created int64
	err := r.Scan(&rm.ID, &rm.Name, &private, &owner, &rm.pinHash, &rm.CapBytes, &rm.RetentionDays, &created)
	if errors.Is(err, sql.ErrNoRows) { return Room{}, ErrNotFound }
	if err != nil{
		return Room{}, err
	}
	rm.Private = private != 0
	rm.OwnerID = owner.Int64
	rm.CreatedAt = fromUnix(created)
	return rm, nil
}

func nullID(id int64) any {
	if id == 0 { return nil }
	return id
}

func (s *Store) RoomByName(name string) (Room, error) {

	return scanRoom(s.db.QueryRow(
		`SELECT `+roomCols+` FROM rooms WHERE name = ?`, NormalizeRoomName(name)))
}


func (s *Store) RoomByID(id int64) (Room, error) {
	return scanRoom(s.db.QueryRow(`SELECT `+roomCols+` FROM rooms WHERE id = ?`, id))
}

func (s *Store) queryRooms(query string, args ...interface{}) ([]Room, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil{
		return nil, err
	}
	defer rows.Close()
	var out []Room
	for rows.Next() {
		r,err := scanRoom(rows)
		if err != nil { return nil, err }
		out = append(out, r)
	}

	return out, rows.Err()
}

func (s *Store) ListPublicRooms() ([]Room, error) {
	return s.queryRooms(`SELECT ` + roomCols +
		` FROM rooms WHERE private = 0 ORDER BY CASE WHEN name = 'lobby' THEN 0 ELSE 1 END, name`)
}
func (s *Store) ListRooms() ([]Room, error) {
	return s.queryRooms(`SELECT ` + roomCols + ` FROM rooms ORDER BY name`)
}

func (s *Store) checkNameFree(name string) error {
	_, err := s.RoomByName(name)
	switch{
	case err == nil:
		return ErrRoomTaken
	case errors.Is(err, ErrNotFound):
		return nil
	default:
		return err
	}
}

func (s *Store) CreatePublicRoom(ownerID int64, name string) (Room, error) {

	name = NormalizeRoomName(name)
	if !ValidRoomName(name) {
		return Room{}, ErrBadRoomName
	}
	if err := s.checkNameFree(name); err != nil { return Room{},err }
	res, err := s.db.Exec(
		`INSERT INTO rooms (name, private, owner_id, retention_days, created_at) VALUES (?, 0, ?, ?, ?)`,
		name, nullID(ownerID), DefaultPublicRetentionDays, s.Now().Unix())
	if err != nil {
		return Room{}, err
	}
	id, err := res.LastInsertId()
	if err != nil { return Room{}, err }
	return s.RoomByID(id)
}


func (s *Store) CreatePrivateRoom(ownerID int64, name, pin string) (Room, error) {
	name = NormalizeRoomName(name)
	if !ValidRoomName(name){
		return Room{}, ErrBadRoomName
	}
	if !ValidPIN(pin) { return Room{},ErrBadPIN }
	owner, err := s.UserByID(ownerID)
	if err != nil {
		return Room{}, err
	}
	var owned int
	err = s.db.QueryRow(
		`SELECT COUNT(*) FROM rooms WHERE owner_id = ? AND private = 1`, ownerID).Scan(&owned)
	if err != nil { return Room{}, err }
	if owned>=owner.MaxPrivateRooms {
		return Room{}, ErrRoomLimit
	}
	if err := s.checkNameFree(name); err != nil { return Room{}, err }
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return Room{},err
	}

	tx, err := s.db.Begin()
	if err != nil { return Room{}, err }
	defer tx.Rollback()
	res, err := tx.Exec(
		`INSERT INTO rooms (name, private, owner_id, pin_hash, cap_bytes, retention_days, created_at)
		 VALUES (?, 1, ?, ?, ?, ?, ?)`,
		name, ownerID, string(hash), DefaultPrivateCapBytes, DefaultPrivateRetentionDays, s.Now().Unix())
	if err != nil {
		return Room{}, err
	}
	id, err := res.LastInsertId()
	if err != nil { return Room{},err }
	if _, err := tx.Exec(`INSERT INTO room_members (room_id, user_id) VALUES (?, ?)`, id, ownerID); err != nil {
		return Room{}, err
	}
	if err := tx.Commit(); err != nil{
		return Room{}, err
	}
	return s.RoomByID(id)
}

func (s *Store) DeleteRoom(name string) error {
	name = NormalizeRoomName(name)
	if name == LobbyName { return ErrProtected }
	room, err := s.RoomByName(name)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil { return err }
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM messages WHERE room_id = ?`,
		`DELETE FROM room_members WHERE room_id = ?`,
	} {
		if _, err := tx.Exec(q, room.ID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM pin_attempts WHERE room_name = ?`, room.Name); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM rooms WHERE id = ?`, room.ID); err != nil { return err }
	return tx.Commit()
}

func (s *Store) SetCap(name string, capBytes int64) error {
	room,err := s.RoomByName(name)
	if err != nil{
		return err
	}
	if !room.Private { return ErrPublicRoom }
	_, err = s.db.Exec(`UPDATE rooms SET cap_bytes = ? WHERE id = ?`, capBytes, room.ID)

	return err
}

func (s *Store) SetRetention(name string, days int) error {
	room, err := s.RoomByName(name)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE rooms SET retention_days = ? WHERE id = ?`, days, room.ID)
	return err
}

func (s *Store) RoomUsage(roomID int64) (int64, error) {
	var used int64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(size), 0) FROM messages WHERE room_id = ?`, roomID).Scan(&used)
	return used, err
}

package store

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	MaxPinFails = 5
	PinLockout  = 15 * time.Minute
)

var (
	dummyOnce sync.Once
	dummyHash []byte
)

func burnTime(pin string) {
	dummyOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("000000"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(pin))
}

func (s *Store) IsMember(roomID, userID int64) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM room_members WHERE room_id = ? AND user_id = ?`, roomID, userID).Scan(&n)
	return n > 0, err
}

func (s *Store) AddMember(roomID, userID int64) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO room_members (room_id, user_id) VALUES (?, ?)`, roomID, userID)
	return err
}

func (s *Store) RemoveMember(roomID, userID int64) error {
	_, err := s.db.Exec(`DELETE FROM room_members WHERE room_id = ? AND user_id = ?`, roomID, userID)
	return err
}

func (s *Store) ChangePIN(roomID int64, pin string) error{
	if !ValidPIN(pin) { return ErrBadPIN }
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE rooms SET pin_hash = ? WHERE id = ? AND private = 1`, string(hash), roomID)
	if err != nil { return err }

	return needRow(res)
}

func (s *Store) JoinPrivate(userID int64, name, pin string) (Room, error) {

	name = NormalizeRoomName(name)
	now := s.Now()

	var fails int
	var lockedUntil int64
	err := s.db.QueryRow(
		`SELECT fails, locked_until FROM pin_attempts WHERE user_id = ? AND room_name = ?`,
		userID, name).Scan(&fails, &lockedUntil)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Room{},err
	}
	if lockedUntil>now.Unix() { return Room{}, &LockedError{Until: time.Unix(lockedUntil, 0)} }
	room, err := s.RoomByName(name)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Room{}, err
	}
	ok := false
	if err == nil && room.Private{
		ok = bcrypt.CompareHashAndPassword([]byte(room.pinHash), []byte(pin)) == nil
	} else {
		burnTime(pin)
	}
	if !ok { return Room{}, s.recordFail(userID, name, fails) }

	_, err = s.db.Exec(`DELETE FROM pin_attempts WHERE user_id = ? AND room_name = ?`, userID, name)
	if err != nil {
		return Room{},err
	}
	if err := s.AddMember(room.ID, userID); err != nil { return Room{}, err }
	return room, nil
}


func (s *Store) recordFail(userID int64, name string, fails int) error {
	fails++
	var locked int64
	if fails>=MaxPinFails {
		locked = s.Now().Add(PinLockout).Unix()
		fails = 0
	}
	_, err := s.db.Exec(
		`INSERT INTO pin_attempts (user_id, room_name, fails, locked_until) VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, room_name) DO UPDATE SET fails = excluded.fails, locked_until = excluded.locked_until`,
		userID, name, fails, locked)
	if err != nil{
		return err
	}
	if locked > 0 { return &LockedError{Until: time.Unix(locked, 0)} }
	return ErrWrongPIN
}

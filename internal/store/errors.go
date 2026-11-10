package store

import (
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrBadNick     = errors.New("nicknames are 3 to 20 characters: letters, digits, _ and -")
	ErrNickTaken   = errors.New("that nickname is taken")
	ErrKeyTaken    = errors.New("that key is already registered")
	ErrBadRoomName = errors.New("room names are 2 to 24 characters: lowercase letters, digits and -")
	ErrRoomTaken = errors.New("that room name is taken")
	ErrBadPIN = errors.New("the PIN must be exactly 6 digits")
	ErrWrongPIN = errors.New("no such room, or wrong PIN")
	ErrRoomFull = errors.New("the room is full")
	ErrRoomLimit = errors.New("private room limit reached")
	ErrProtected = errors.New("the lobby cannot be changed that way")
	ErrPublicRoom = errors.New("that room is public; only private rooms have a size limit")
)

type LockedError struct{ Until time.Time }

func (e *LockedError) Error() string {
	return "too many wrong PINs; locked until " + e.Until.Format(time.RFC3339)
}

type CooldownError struct{ Until time.Time }

func (e *CooldownError) Error() string {
	return "nickname changed recently; next change allowed at " + e.Until.Format(time.RFC3339)
}

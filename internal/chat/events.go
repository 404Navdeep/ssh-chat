package chat

import (
	"time"

	"sshchat/internal/store"
)

type EventKind int

const (
	EvMessage EventKind = iota
	EvSystem
	EvNotice
	EvError
	EvDM
	EvRoom
	EvRooms
	EvClose
)

type Event struct {
	Kind    EventKind
	Room    string
	Private bool
	Nick string
	Body string
	At time.Time
	Out bool
	History []store.Message
}

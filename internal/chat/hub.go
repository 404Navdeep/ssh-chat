### internal/chat/hub.go
package chat

import (
	"errors"
	"log"
	"sync"
	"time"

	"sshchat/internal/store"
)

const eventBuffer = 512

var ErrBanned = errors.New("this account is banned")

type Hub struct {
	st *store.Store
	Now func() time.Time
	Logf func(format string, args ...any)

	mu sync.Mutex
	sessions map[*Session]struct{}
	limiters map[int64]*limiter
}

type Session struct {
	hub    *Hub
	Events chan Event

	user store.User
	room store.Room
	closed bool
}

func NewHub(st *store.Store) *Hub {
	return &Hub{
		st:       st,
		Now:      time.Now,
		Logf:     log.Printf,
		sessions: make(map[*Session]struct{}),
		limiters: make(map[int64]*limiter),
	}
}

func (s *Session) User() store.User{
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	return s.user
}

func (s *Session) Room() store.Room {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	return s.room
}


func (h *Hub) Connect(u store.User) (*Session, error) {
	if u.Banned {
		return nil, ErrBanned
	}
	lobby, err := h.st.RoomByName(store.LobbyName)
	if err != nil { return nil,err }
	s := &Session{hub: h, Events: make(chan Event, eventBuffer), user: u}
	h.mu.Lock()
	h.sessions[s] = struct{}{}
	h.mu.Unlock()
	h.enterRoom(s, lobby)
	h.deliverPending(s, u.ID)

	return s, nil
}

func (h *Hub) Disconnect(s *Session) {

	h.mu.Lock()
	defer h.mu.Unlock()
	if s.closed{
		return
	}
	s.closed = true
	delete(h.sessions, s)
	close(s.Events)
	if s.room.ID != 0 {
		h.pushRoom(s.room.ID, nil, Event{
			Kind: EvSystem,Room: s.room.Name,At: h.Now(),
			Body: s.user.Nick + " left #" + s.room.Name,
		})
	}
}

func (h *Hub) enterRoom(s *Session, room store.Room) {
	history, err := h.st.RecentMessages(room.ID, HistoryLimit)
	if err != nil {
		h.Logf("history of #%s: %v", room.Name, err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if s.closed { return }
	old := s.room
	s.room = room
	now := h.Now()
	if old.ID != 0 {
		h.pushRoom(old.ID, s, Event{
			Kind: EvSystem, Room: old.Name, At: now, Body: s.user.Nick + " left #" + old.Name,
		})
	}
	h.pushRoom(room.ID, s, Event{
		Kind: EvSystem, Room: room.Name, At: now, Body: s.user.Nick + " joined #" + room.Name,
	})
	h.push(s, Event{Kind: EvRoom, Room: room.Name, Private: room.Private, History: history, At: now})
}
func (h *Hub) goLobby(s *Session){
	lobby, err := h.st.RoomByName(store.LobbyName)
	if err != nil {
		h.Logf("lobby: %v", err)
		return
	}
	h.enterRoom(s, lobby)
}

func (h *Hub) deliverPending(s *Session, userID int64) {
	dms,err := h.st.TakePendingDMs(userID,PendingDMLimit)
	if err != nil {
		h.Logf("pending DMs of user %d: %v", userID, err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, d := range dms {
		h.push(s, Event{Kind: EvDM, Nick: d.FromNick, Body: d.Body, At: d.At})
	}
}

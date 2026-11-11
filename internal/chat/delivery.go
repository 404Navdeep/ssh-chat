### internal/chat/delivery.go
package chat

func (h *Hub) push(s *Session, ev Event){
	if s.closed {
		return
	}
	select {
	case s.Events <- ev:
	default:
	}
}

func (h *Hub) pushRoom(roomID int64, except *Session, ev Event) {
	for s := range h.sessions {
		if s != except && s.room.ID==roomID {
			h.push(s, ev)
		}
	}
}


func (h *Hub) tell(s *Session, ev Event){

	h.mu.Lock()
	defer h.mu.Unlock()
	h.push(s, ev)
}

func (h *Hub) say(s *Session, text string) {
	h.tell(s, Event{Kind: EvSystem, Body: text, At: h.Now()})
}

func (h *Hub) notice(s *Session, text string) {
	h.tell(s, Event{Kind: EvNotice, Body: text, At: h.Now()})
}

func (h *Hub) fail(s *Session, text string) {
	h.tell(s,Event{Kind: EvError,Body: text,At: h.Now()})
}

func (h *Hub) toRoom(roomID int64, ev Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pushRoom(roomID, nil, ev)
}

func (h *Hub) toUser(userID int64, ev Event) int{
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for s := range h.sessions {
		if s.user.ID == userID && !s.closed {
			h.push(s, ev)
			n++
		}
	}
	return n
}
func (h *Hub) toAll(ev Event) {

	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.sessions {
		h.push(s, ev)
	}
}

func (h *Hub) snapshot() []*Session{
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*Session, 0, len(h.sessions))
	for s := range h.sessions {
		out = append(out, s)
	}
	return out
}

func (h *Hub) sessionsOf(userID int64) []*Session {

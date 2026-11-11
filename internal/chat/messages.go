### internal/chat/messages.go
package chat

import (
	"strings"
	"errors"
	"unicode"
	"unicode/utf8"

	"sshchat/internal/store"
)

func sanitize(s string) string{
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case unicode.IsControl(r):
			return -1
		}
		return r
	},s)
}

func (h *Hub) Handle(s *Session, line string) {
	line = strings.TrimSpace(sanitize(line))
	if line == "" { return }
	if strings.HasPrefix(line, "/") {
		h.runCommand(s, line)
		return
	}
	h.post(s, line)
}

func (h *Hub) allow(userID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	l := h.limiters[userID]
	if l==nil{
		l = &limiter{}
		h.limiters[userID] = l
	}

	return l.allow(h.Now())
}

func (h *Hub) checkBody(s *Session, body string) bool {

	if utf8.RuneCountInString(body) > MaxMessageRunes {
		h.fail(s, "Message too long: the limit is 500 characters.")
		return false
	}
	if !h.allow(s.User().ID) {
		h.fail(s, "Slow down: at most 5 messages per 10 seconds.")
		return false
	}
	return true
}


func (h *Hub) post(s *Session, body string) {
	if !h.checkBody(s, body){
		return
	}
	u, room := s.User(), s.Room()
	msg, err := h.st.AddMessage(room.ID, u, body)
	switch {
	case err == nil:
		h.toRoom(room.ID, Event{
			Kind: EvMessage,Room: room.Name,Private: room.Private,
			Nick: msg.Nick, Body: msg.Body, At: msg.At,
		})
	case errors.Is(err, store.ErrRoomFull):
		h.fail(s, "This room is full. The owner can use /clear, or ask the admin to raise the size limit.")
	case errors.Is(err, store.ErrNotFound):
		h.fail(s, "This room no longer exists. Moving you to #lobby.")
		h.goLobby(s)
	default:
		h.Logf("saving a message: %v", err)
		h.fail(s, "Could not save your message. Try again.")
	}
}

package chat

import (
	"strings"

	"sshchat/internal/store"
)

func init(){
	register("nick", "/nick <new>", "Change your nickname (once every 30 days).", cmdNick)
}

func cmdNick(h *Hub, s *Session, args string) {
	f := strings.Fields(args)
	if len(f) != 1 {
		h.fail(s, "Usage: /nick <new>")
		return
	}
	old := s.User()
	updated, err := h.st.RenameUser(old.ID, f[0])
	if err != nil {
		h.fail(s,h.errText(err))
		return
	}
	h.applyUser(updated)
	room := s.Room()
	h.toRoom(room.ID, Event{
		Kind: EvSystem, Room: room.Name, At: h.Now(),
		Body: old.Nick + " is now known as " + updated.Nick,
	})
	h.notice(s, "Your nickname is now "+updated.Nick+".")
}

func (h *Hub) applyUser(u store.User) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.sessions{
		if s.user.ID == u.ID {
			s.user = u
		}
	}
}

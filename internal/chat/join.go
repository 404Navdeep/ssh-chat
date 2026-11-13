package chat

import (
	"errors"
	"strings"

	"sshchat/internal/store"
)

func init(){

	register("join", "/join <room> [pin]", "Join a room. Private rooms need their 6-digit PIN.", cmdJoin)
}

func (h *Hub) moveTo(s *Session, room store.Room) {
	if s.Room().ID == room.ID {
		h.notice(s, "You are already in #"+room.Name+".")
		return
	}
	h.enterRoom(s, room)
}


func cmdJoin(h *Hub, s *Session, args string) {
	f := strings.Fields(args)
	if len(f) == 0 {
		h.fail(s, "Usage: /join <room> [pin]")
		return
	}
	name := store.NormalizeRoomName(f[0])
	pin := ""
	if len(f)>1{
		pin = f[1]
	}
	u := s.User()

	room, err := h.st.RoomByName(name)
	switch {
	case err == nil && !room.Private:
		h.moveTo(s, room)
		return
	case err == nil:
		member, merr := h.st.IsMember(room.ID, u.ID)
		if merr==nil && member {
			h.moveTo(s, room)
			return
		}
	case !errors.Is(err,store.ErrNotFound):
		h.fail(s, h.errText(err))
		return
	}
	if pin == "" {
		h.fail(s, h.errText(store.ErrWrongPIN))
		return
	}
	joined, err := h.st.JoinPrivate(u.ID, name, pin)
	if err != nil {
		h.Logf("failed join: user=%s room=%s: %v", u.Nick, name, err)
		h.fail(s, h.errText(err))
		return
	}
	h.moveTo(s, joined)
}

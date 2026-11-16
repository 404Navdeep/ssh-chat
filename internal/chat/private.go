package chat

import (
	"fmt"
	"strings"

	"sshchat/internal/store"
)

func init() {
	register("create-private", "/create-private <room> <pin>",
		"Create a private room. People join with its name and 6-digit PIN.", cmdCreatePrivate)
	register("pin", "/pin <new pin>", "Owner: change the PIN of this room.", cmdPin)
	register("remove", "/remove <nick>", "Owner: remove a member from this room.", cmdRemove)
	register("clear", "/clear", "Owner: delete the saved messages of this room.", cmdClear)
}

func cmdCreatePrivate(h *Hub, s *Session, args string) {
	f := strings.Fields(args)
	if len(f) != 2 {
		h.fail(s, "Usage: /create-private <room> <pin>")
		return
	}
	room, err := h.st.CreatePrivateRoom(s.User().ID, f[0], f[1])
	if err != nil {
		h.fail(s, h.errText(err))
		return
	}
	h.enterRoom(s, room)
	h.say(s, fmt.Sprintf("Private room #%s created. People join with: /join %s <pin>", room.Name, room.Name))
	h.say(s, fmt.Sprintf("It keeps up to %d MB of messages, for %d days.", room.CapBytes>>20, room.RetentionDays))
}

func (h *Hub) ownedRoom(s *Session) (store.Room, bool){
	room, err := h.st.RoomByID(s.Room().ID)
	if err != nil {
		h.fail(s, h.errText(err))
		return store.Room{}, false
	}
	if !room.Private || room.OwnerID == 0 || room.OwnerID != s.User().ID {
		h.fail(s, "Only the owner of a private room can do that.")
		return store.Room{},false
	}
	return room, true
}

func cmdPin(h *Hub, s *Session, args string) {
	pin := firstField(args)
	if pin == "" {
		h.fail(s, "Usage: /pin <new pin>")
		return
	}
	room, ok := h.ownedRoom(s)
	if !ok { return }
	if err := h.st.ChangePIN(room.ID, pin); err != nil{
		h.fail(s, h.errText(err))
		return
	}
	h.notice(s, "PIN changed. Current members stay in; the new PIN is for people joining later.")
}

func cmdRemove(h *Hub, s *Session, args string) {
	nick := firstField(args)
	if nick == "" {
		h.fail(s, "Usage: /remove <nick>")
		return
	}
	room, ok := h.ownedRoom(s)
	if !ok {
		return
	}
	target, err := h.st.UserByNick(nick)
	if err != nil {
		h.fail(s, "No such user.")
		return
	}
	if target.ID == room.OwnerID{
		h.fail(s, "You cannot remove yourself.")
		return
	}
	member,err := h.st.IsMember(room.ID,target.ID)
	if err != nil {
		h.fail(s, h.errText(err))
		return
	}
	if !member {
		h.fail(s, target.Nick+" is not a member of this room.")
		return
	}
	if err := h.st.RemoveMember(room.ID, target.ID); err != nil {
		h.fail(s, h.errText(err))
		return
	}
	h.kickFromRoom(target.ID, room.ID, "You were removed from #"+room.Name+" by the owner.")
	h.notice(s, target.Nick+" was removed from #"+room.Name+".")
}

func cmdClear(h *Hub, s *Session, _ string) {

	room, ok := h.ownedRoom(s)
	if !ok { return }
	if err := h.st.ClearRoom(room.ID); err != nil{
		h.fail(s, h.errText(err))
		return
	}
	now := h.Now()
	h.toRoom(room.ID, Event{Kind: EvRoom, Room: room.Name, Private: true, At: now})
	h.toRoom(room.ID, Event{Kind: EvSystem, Room: room.Name, At: now, Body: "The owner cleared the room."})
}


func (h *Hub) kickFromRoom(userID, roomID int64, reason string) {
	for _,sess := range h.sessionsOf(userID) {
		if sess.Room().ID != roomID {
			continue
		}
		h.fail(sess, reason)
		h.goLobby(sess)
	}
}

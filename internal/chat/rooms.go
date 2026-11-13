package chat

import (
	"sort"
	"fmt"
	"strings"

	"sshchat/internal/store"
)

type RoomInfo struct {
	Name   string
	Online int
}

func (h *Hub) RoomList() []RoomInfo {
	rooms, err := h.st.ListPublicRooms()
	if err != nil {
		h.Logf("room list: %v", err)
		return nil
	}
	type pair struct{ room, user int64 }
	seen := make(map[pair]bool)
	online := make(map[int64]int)
	h.mu.Lock()
	for s := range h.sessions {
		p := pair{s.room.ID, s.user.ID}
		if !seen[p] {
			seen[p] = true
			online[s.room.ID]++
		}
	}
	h.mu.Unlock()
	out := make([]RoomInfo,0,len(rooms))
	for _, r := range rooms{
		out = append(out, RoomInfo{Name: r.Name, Online: online[r.ID]})
	}
	return out
}

func (h *Hub) Who(roomID int64) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	seen := make(map[string]bool)
	var names []string
	for s := range h.sessions {
		if s.room.ID == roomID && !seen[s.user.Nick] {
			seen[s.user.Nick] = true
			names = append(names, s.user.Nick)
		}
	}
	sort.Strings(names)

	return names
}

func firstField(args string) string {

	f := strings.Fields(args)
	if len(f) == 0{
		return ""
	}
	return f[0]
}


func init() {
	register("rooms", "/rooms", "List the public rooms.", cmdRooms)
	register("who", "/who", "Show who is in this room.", cmdWho)
	register("leave", "/leave", "Go back to #lobby.", cmdLeave)
	register("create", "/create <room>", "Create a public room.", cmdCreate)
}

func cmdRooms(h *Hub, s *Session, _ string) {
	h.say(s, "Public rooms:")
	for _, r := range h.RoomList() {
		h.say(s, fmt.Sprintf("  #%s (%d online)", r.Name, r.Online))
	}
}

func cmdWho(h *Hub, s *Session, _ string) {
	room := s.Room()
	h.say(s, fmt.Sprintf("In #%s: %s", room.Name, strings.Join(h.Who(room.ID), ", ")))
}
func cmdLeave(h *Hub, s *Session, _ string){
	if s.Room().Name==store.LobbyName {
		h.fail(s, "You are in #lobby already. There is nowhere to leave to.")
		return
	}
	h.goLobby(s)
}

func cmdCreate(h *Hub, s *Session, args string) {
	name := firstField(args)
	if name == "" {
		h.fail(s, "Usage: /create <room>")
		return
	}
	room, err := h.st.CreatePublicRoom(s.User().ID, name)
	if err != nil {
		h.fail(s, h.errText(err))
		return
	}
	h.toAll(Event{Kind: EvRooms,At: h.Now()})
	h.enterRoom(s, room)
}

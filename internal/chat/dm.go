package chat

import (
	"errors"
	"strings"

	"sshchat/internal/store"
)

func init() {
	register("msg", "/msg <nick> <text>", "Send a direct message.", cmdMsg)
}

func cmdMsg(h *Hub, s *Session, args string){
	nick, text, _ := strings.Cut(args, " ")
	text = strings.TrimSpace(text)
	if nick == "" || text == "" {
		h.fail(s, "Usage: /msg <nick> <text>")
		return
	}
	me := s.User()
	target, err := h.st.UserByNick(nick)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			h.Logf("looking up %s: %v", nick, err)
		}
		h.fail(s, "No such user.")
		return
	}
	if target.Banned {
		h.fail(s, "No such user.")
		return
	}
	if target.ID==me.ID{
		h.fail(s, "You cannot message yourself.")
		return
	}
	if !h.checkBody(s, text) { return }
	now := h.Now()
	delivered := h.toUser(target.ID, Event{Kind: EvDM, Nick: me.Nick, Body: text, At: now})
	if delivered == 0 {
		if err := h.st.AddDM(me.Nick, target.ID, text); err != nil {
			h.Logf("storing a direct message: %v", err)
			h.fail(s, "Could not send your message. Try again.")
			return
		}
		h.notice(s, target.Nick+" is offline. They will see your message when they log in (kept 30 days).")
	}
	h.toUser(me.ID,Event{Kind: EvDM,Nick: target.Nick,Body: text,At: now,Out: true})
}

package chat

import (
	"strings"
	"testing"
)

func TestDMToAnOnlineUser(t *testing.T) {

	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	drain(alice)
	drain(bob)

	e.hub.Handle(alice, "/msg bobby hello bob")
	got := drain(bob)
	if len(got) != 1 || got[0].Kind != EvDM || got[0].Nick != "alice" || got[0].Body != "hello bob" || got[0].Out{
		t.Errorf("bobby got %+v", got)
	}
	echo := drain(alice)
	if len(echo) != 1 || echo[0].Kind != EvDM || echo[0].Nick != "bobby" || !echo[0].Out {
		t.Errorf("alice got %+v", echo)
	}
	if pending, _ := e.st.TakePendingDMs(bob.User().ID, 10); len(pending) != 0 {
		t.Error("a delivered DM was also stored")
	}
}

func TestDMToAnOfflineUser(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	bobUser := e.user("bobby")
	drain(alice)

	e.hub.Handle(alice, "/msg bobby are you there?")
	if !has(drain(alice), EvNotice, "offline") {
		t.Error("alice was not told bobby is offline")
	}

	bob, err := e.hub.Connect(bobUser)
	if err != nil{
		t.Fatal(err)
	}
	var found bool
	for _, ev := range drain(bob) {
		if ev.Kind == EvDM && ev.Nick == "alice" && ev.Body == "are you there?" {
			found = true
		}
	}
	if !found {
		t.Error("bobby did not get the stored DM at login")
	}
}

func TestDMReachesEverySessionOfAUser(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	bobUser := e.user("bobby")
	first, _ := e.hub.Connect(bobUser)
	second,_ := e.hub.Connect(bobUser)
	drain(alice)
	drain(first)
	drain(second)
	e.hub.Handle(alice, "/msg bobby hi both")
	if count(drain(first), EvDM) != 1 || count(drain(second), EvDM) != 1{
		t.Error("both terminals of bobby should get the DM")
	}
}

func TestDMErrors(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	e.user("carol")
	e.user("dave")
	if err := e.st.SetBanned("carol", true); err != nil {
		t.Fatal(err)
	}
	drain(alice)

	for input, want := range map[string]string{
		"/msg": "Usage",
		"/msg dave": "Usage",
		"/msg ghost hi": "No such user",
		"/msg carol hi": "No such user",
		"/msg alice hi": "yourself",
		"/msg dave " + strings.Repeat("a", 501): "too long",
	} {
		e.hub.Handle(alice, input)
		if !has(drain(alice), EvError, want) {
			t.Errorf("%.30q: no error containing %q", input, want)
		}
	}
}
func TestDMsCountTowardTheRateLimit(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	e.user("bobby")
	drain(alice)

	for i := 0; i < RateMessages; i++{
		e.hub.Handle(alice, "filler")
	}
	drain(alice)
	e.hub.Handle(alice, "/msg bobby too fast")
	if !has(drain(alice), EvError, "Slow down") {
		t.Error("a DM got around the rate limit")
	}
}

### internal/chat/hub_test.go
package chat

import (
	"strings"
	"errors"
	"testing"
	"time"
)

func TestConnectEntersLobby(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	evs := drain(alice)
	if len(evs) != 1 || evs[0].Kind != EvRoom || evs[0].Room != "lobby" || evs[0].Private{
		t.Fatalf("events = %+v", evs)
	}
	if alice.Room().Name != "lobby" || alice.User().Nick != "alice" {
		t.Errorf("session state: room %q, user %q", alice.Room().Name, alice.User().Nick)
	}
}

func TestConnectRejectsBannedUser(t *testing.T) {
	e := newEnv(t)
	e.user("alice")
	if err := e.st.SetBanned("alice", true); err != nil {
		t.Fatal(err)
	}
	u, _ := e.st.UserByNick("alice")
	if _, err := e.hub.Connect(u); !errors.Is(err, ErrBanned) {
		t.Errorf("Connect error = %v, want ErrBanned", err)
	}
}

func TestOthersSeeJoinsAndLeaves(t *testing.T){
	e := newEnv(t)
	alice := e.connect("alice")
	drain(alice)
	bob := e.connect("bobby")
	if !has(drain(alice), EvSystem, "bobby joined #lobby") {
		t.Error("alice did not see bobby join")
	}
	drain(bob)
	e.hub.Disconnect(bob)
	if !has(drain(alice), EvSystem, "bobby left #lobby") {
		t.Error("alice did not see bobby leave")
	}
}

func TestPostReachesEveryoneInTheRoom(t *testing.T) {

	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	drain(alice)
	drain(bob)
	e.hub.Handle(alice, "hello there")

	for name, s := range map[string]*Session{"alice": alice, "bobby": bob} {
		evs := drain(s)
		if len(evs) != 1 || evs[0].Kind != EvMessage || evs[0].Nick != "alice" ||
			evs[0].Body != "hello there" || evs[0].Room != "lobby" {
			t.Errorf("%s got %+v", name, evs)
		}
	}
}

func TestHistoryIsShownOnConnect(t *testing.T){
	e := newEnv(t)
	alice := e.connect("alice")
	e.hub.Handle(alice, "one")
	e.hub.Handle(alice, "two")

	carol := e.connect("carol")
	evs := drain(carol)
	if len(evs) == 0 || evs[0].Kind != EvRoom || len(evs[0].History) != 2 ||
		evs[0].History[0].Body != "one" || evs[0].History[1].Body != "two" {
		t.Errorf("events = %+v", evs)
	}
}


func TestEmptyLineIsIgnored(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	drain(alice)
	drain(bob)
	e.hub.Handle(alice, "   ")
	if got := drain(bob); len(got) != 0 {
		t.Errorf("bobby got %+v", got)
	}
}

func TestControlCharactersAreRemoved(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	drain(alice)
	drain(bob)
	e.hub.Handle(alice, "\x1b[31mred\x07\tend")
	evs := drain(bob)
	if len(evs) != 1 || evs[0].Body != "[31mred end" {
		t.Errorf("bobby got %+v", evs)
	}
}

func TestMessageLengthLimit(t *testing.T){
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	drain(alice)
	drain(bob)
	e.hub.Handle(alice, strings.Repeat("a", 501))
	if !has(drain(alice), EvError, "too long") {
		t.Error("alice was not told her message is too long")
	}
	if got := drain(bob); len(got) != 0 {
		t.Errorf("bobby got %+v", got)
	}

	e.hub.Handle(alice, strings.Repeat("é", 500))
	if count(drain(bob), EvMessage) != 1 {
		t.Error("a 500-character message was refused")
	}
}
func TestRateLimit(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	drain(alice)
	drain(bob)

	for i := 0; i<RateMessages; i++{
		e.hub.Handle(alice, "message")
	}
	e.hub.Handle(alice, "one too many")
	if !has(drain(alice), EvError, "Slow down") {
		t.Error("alice was not told to slow down")
	}
	if n := count(drain(bob), EvMessage); n != RateMessages {
		t.Errorf("bobby got %d messages, want %d", n, RateMessages)
	}
	e.clk.Advance(RateWindow + time.Second)
	e.hub.Handle(alice, "back again")
	if count(drain(bob), EvMessage) != 1 {
		t.Error("alice could not post after waiting")
	}
}

func TestLimiterWindow(t *testing.T) {

	var l limiter
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < RateMessages; i++{
		if !l.allow(now) {
			t.Fatalf("hit %d refused", i+1)
		}
	}
	if l.allow(now.Add(5*time.Second)) {
		t.Error("sixth hit inside the window was allowed")
	}
	if !l.allow(now.Add(RateWindow + time.Second)) {
		t.Error("hit after the window was refused")
	}
}

func TestPendingDMsAreDeliveredOnConnect(t *testing.T) {
	e := newEnv(t)
	bob := e.user("bobby")
	if err := e.st.AddDM("alice", bob.ID, "while you were out"); err != nil{
		t.Fatal(err)
	}
	s,err := e.hub.Connect(bob)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, ev := range drain(s) {
		if ev.Kind == EvDM && ev.Nick == "alice" && ev.Body == "while you were out" && !ev.Out {
			found = true
		}
	}
	if !found {
		t.Error("the stored DM was not delivered")
	}
	e.hub.Disconnect(s)
	again, _ := e.hub.Connect(bob)
	if count(drain(again), EvDM) != 0{
		t.Error("the DM was delivered twice")
	}
}

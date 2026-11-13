package chat

import (
	"strings"
	"testing"
	"time"
)

func TestCreateJoinChatAndLeave(t *testing.T) {

	e := newEnv(t)
	alice, bob, carol := e.connect("alice"), e.connect("bobby"), e.connect("carol")
	drain(alice)
	drain(bob)
	drain(carol)

	e.hub.Handle(alice, "/create go")
	aliceEvs := drain(alice)
	if count(aliceEvs, EvRoom) != 1 || aliceEvs[len(aliceEvs)-1].Room != "go" || alice.Room().Name != "go" {
		t.Fatalf("alice events after /create: %+v", aliceEvs)
	}
	if count(drain(bob), EvRooms) != 1 {
		t.Error("bobby was not told the room list changed")
	}
	drain(carol)
	e.hub.Handle(bob, "/rooms")
	listing := drain(bob)
	if !has(listing, EvSystem, "#go (1 online)") || !has(listing, EvSystem, "#lobby (2 online)"){
		t.Errorf("/rooms output = %+v", listing)
	}

	e.hub.Handle(bob, "/join #Go")
	if !has(drain(alice), EvSystem, "bobby joined #go") {
		t.Error("alice did not see bobby join #go")
	}
	if bob.Room().Name != "go" {
		t.Fatalf("bobby is in %q", bob.Room().Name)
	}
	drain(bob)

	e.hub.Handle(bob, "/who")
	if !has(drain(bob), EvSystem, "In #go: alice, bobby") {
		t.Error("/who output is wrong")
	}
	e.hub.Handle(bob, "hi go")
	if !has(drain(alice), EvMessage, "hi go") {
		t.Error("alice did not get the message")
	}
	if count(drain(carol), EvMessage) != 0{
		t.Error("carol, who is in #lobby, got a message from #go")
	}

	e.hub.Handle(bob, "/leave")
	if bob.Room().Name != "lobby" {
		t.Errorf("bobby is in %q after /leave", bob.Room().Name)
	}
	if !has(drain(alice), EvSystem, "bobby left #go") {
		t.Error("alice did not see bobby leave")
	}
}

func TestLeavingTheLobbyIsAnError(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	drain(alice)
	e.hub.Handle(alice, "/leave")
	if !has(drain(alice), EvError, "nowhere to leave") {
		t.Error("alice was not told she cannot leave the lobby")
	}
}

func TestCreateErrors(t *testing.T){
	e := newEnv(t)
	alice := e.connect("alice")
	drain(alice)
	for input, want := range map[string]string{
		"/create": "Usage",
		"/create a": "2 to 24",
		"/create lobby": "taken",
	} {
		e.hub.Handle(alice, input)
		if !has(drain(alice),EvError,want) {
			t.Errorf("%q: no error containing %q", input, want)
		}
	}
}

func TestJoinUnknownRoom(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	drain(alice)
	e.hub.Handle(alice, "/join ghost")
	e.hub.Handle(alice, "/join ghost 123456")
	e.hub.Handle(alice, "/join")
	evs := drain(alice)
	if count(evs, EvError) != 3 || !has(evs, EvError, "No such room, or wrong PIN") || !has(evs, EvError, "Usage") {
		t.Errorf("events = %+v", evs)
	}
}
func TestJoinTheRoomYouAreIn(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	drain(alice)
	e.hub.Handle(alice, "/join lobby")
	if !has(drain(alice), EvNotice, "already in #lobby"){
		t.Error("no 'already in' notice")
	}
}

func TestJoinPrivateRoomWithPIN(t *testing.T) {
	e := newEnv(t)
	owner := e.user("owner")
	if _, err := e.st.CreatePrivateRoom(owner.ID, "secret", "123456"); err != nil {
		t.Fatal(err)
	}
	bob := e.connect("bobby")
	drain(bob)

	e.hub.Handle(bob, "/join secret")
	if !has(drain(bob), EvError, "No such room, or wrong PIN") {
		t.Error("joining without a PIN should fail")
	}
	e.hub.Handle(bob, "/join secret 000000")
	if !has(drain(bob), EvError, "No such room, or wrong PIN") {
		t.Error("a wrong PIN should fail")
	}
	if !e.logged("failed join"){
		t.Error("the failed attempt was not logged")
	}
	for _, line := range e.logs {
		if strings.Contains(line, "000000") || strings.Contains(line, "123456") {
			t.Errorf("a PIN was written to the log: %q", line)
		}
	}

	e.hub.Handle(bob, "/join SECRET 123456")
	evs := drain(bob)
	if count(evs, EvRoom) != 1 || evs[len(evs)-1].Room != "secret" || !evs[len(evs)-1].Private {
		t.Fatalf("events after the right PIN: %+v", evs)
	}

	e.hub.Handle(bob, "/leave")
	drain(bob)
	e.hub.Handle(bob, "/join secret")
	if bob.Room().Name != "secret" {
		t.Errorf("member could not rejoin; room = %q", bob.Room().Name)
	}
}

func TestPinLockoutThroughTheChat(t *testing.T){

	e := newEnv(t)
	owner := e.user("owner")
	if _, err := e.st.CreatePrivateRoom(owner.ID, "secret", "123456"); err != nil {
		t.Fatal(err)
	}
	bob := e.connect("bobby")
	drain(bob)
	for i := 0; i < 4; i++ {
		e.hub.Handle(bob, "/join secret 000000")
	}
	if has(drain(bob), EvError, "Too many") {
		t.Fatal("locked out too early")
	}
	e.hub.Handle(bob, "/join secret 000000")
	if !has(drain(bob), EvError, "Try again in 15 minutes") {
		t.Fatal("no lockout message on the fifth wrong PIN")
	}
	e.hub.Handle(bob, "/join secret 123456")
	if !has(drain(bob), EvError, "Too many wrong PINs") || bob.Room().Name != "lobby"{
		t.Error("the right PIN worked during the lockout")
	}

	e.clk.Advance(16*time.Minute)
	e.hub.Handle(bob, "/join secret 123456")
	if bob.Room().Name != "secret" {
		t.Errorf("could not join after the lockout; room = %q", bob.Room().Name)
	}
}

func TestNickCommand(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	drain(alice)
	drain(bob)

	e.hub.Handle(alice, "/nick alicia")
	if alice.User().Nick != "alicia" {
		t.Fatalf("nick = %q", alice.User().Nick)
	}
	if !has(drain(bob), EvSystem, "alice is now known as alicia") {
		t.Error("bobby was not told about the new nickname")
	}
	if !has(drain(alice), EvNotice, "alicia"){
		t.Error("alice got no confirmation")
	}
	e.hub.Handle(alice, "/nick alice2")
	if !has(drain(alice), EvError, "again in 30 days") {
		t.Error("the cooldown was not enforced")
	}

	e.clk.Advance(31 * 24 * time.Hour)
	e.hub.Handle(alice, "/nick bobby")
	e.hub.Handle(alice, "/nick x")
	e.hub.Handle(alice, "/nick")
	evs := drain(alice)
	if !has(evs, EvError, "taken") || !has(evs, EvError, "3 to 20") || !has(evs, EvError, "Usage") {
		t.Errorf("events = %+v", evs)
	}

	e.hub.Handle(alice, "/nick alice2")
	drain(alice)
	drain(bob)
	e.hub.Handle(alice, "hello")
	msgs := drain(bob)
	if len(msgs) != 1 || msgs[0].Nick != "alice2" {
		t.Errorf("message after rename = %+v", msgs)
	}
}

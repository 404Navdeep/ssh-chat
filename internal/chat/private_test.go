func TestCreatePrivateRoomAndTalk(t *testing.T) {
	e := newEnv(t)
	alice, bob, carol := e.connect("alice"), e.connect("bobby"), e.connect("carol")
	drain(alice)
	drain(bob)
	drain(carol)

	e.hub.Handle(alice, "/create-private secret 123456")
	evs := drain(alice)
	if count(evs, EvRoom) != 1 || !evs[0].Private || evs[0].Room != "secret" ||
		!has(evs, EvSystem, "Private room #secret created") || !has(evs, EvSystem, "10 MB") {
		t.Fatalf("events after /create-private: %+v", evs)
		}
	room, err := e.st.RoomByName("secret")
	if err != nil || !room.Private || room.OwnerID != alice.User().ID {
		t.Fatalf("stored room = %+v, %v", room, err)
	}
	if has(drain(bob), EvSystem, "secret") || has(drain(carol), EvSystem, "secret"){
		t.Error("the private room was announced")
	}
	e.hub.Handle(carol, "/rooms")
	if has(drain(carol), EvSystem, "secret") {
		t.Error("the private room is listed in /rooms")
	}

	e.hub.Handle(bob, "/join secret 123456")
	if !has(drain(alice), EvSystem, "bobby joined #secret") {
		t.Error("alice did not see bobby join")
	}
	drain(bob)
	e.hub.Handle(bob, "psst")
	if !has(drain(alice), EvMessage, "psst") {
		t.Error("alice did not get the message")
	}
	if count(drain(carol), EvMessage) != 0 {
		t.Error("carol saw a private message")
	}
	msgs, _ := e.st.RecentMessages(room.ID, 10)
	if len(msgs) != 1 || msgs[0].Body != "psst"{
		t.Errorf("saved messages = %+v", msgs)
	}
}

func TestCreatePrivateRoomErrors(t *testing.T) {
	e := newEnv(t)
	alice := e.connect("alice")
	drain(alice)
	for input, want := range map[string]string{
		"/create-private":                "Usage",
		"/create-private only-a-name":    "Usage",
		"/create-private room-a 12345":   "6 digits",
		"/create-private room-a abcdef":  "6 digits",
		"/create-private a 123456":       "2 to 24",
		"/create-private lobby 123456":   "taken",
	} {
		e.hub.Handle(alice, input)
		if !has(drain(alice), EvError, want) {
			t.Errorf("%q: no error containing %q", input, want)
		}
	}
}
func TestPrivateRoomLimit(t *testing.T) {

	e := newEnv(t)
	alice := e.connect("alice")
	drain(alice)
	for _, name := range []string{"room-a", "room-b", "room-c"} {
		e.hub.Handle(alice, "/create-private "+name+" 123456")
	}
	drain(alice)
	e.hub.Handle(alice, "/create-private room-d 123456")
	if !has(drain(alice), EvError, "private room limit"){
		t.Error("the fourth private room was allowed")
	}
	if err := e.st.SetMaxPrivateRooms("alice", 4); err != nil {
		t.Fatal(err)
	}
	e.hub.Handle(alice, "/create-private room-d 123456")
	if alice.Room().Name != "room-d" {
		t.Errorf("alice is in %q after the limit was raised", alice.Room().Name)
	}
}

func TestOwnerChangesPIN(t *testing.T) {
	e := newEnv(t)
	alice, bob, carol := e.connect("alice"), e.connect("bobby"), e.connect("carol")
	e.hub.Handle(alice, "/create-private secret 123456")
	e.hub.Handle(bob, "/join secret 123456")
	drain(alice)
	drain(bob)
	drain(carol)
	e.hub.Handle(alice, "/pin 654321")
	if !has(drain(alice), EvNotice, "PIN changed") {
		t.Fatal("no confirmation for /pin")
	}

	e.hub.Handle(bob, "/leave")
	e.hub.Handle(bob, "/join secret")
	if bob.Room().Name != "secret"{
		t.Errorf("member lost access after the PIN change; room = %q", bob.Room().Name)
	}

	e.hub.Handle(carol, "/join secret 123456")
	if carol.Room().Name == "secret" {
		t.Error("the old PIN still works")
	}
	e.hub.Handle(carol, "/join secret 654321")
	if carol.Room().Name != "secret" {
		t.Error("the new PIN does not work")
	}
	drain(alice)
	e.hub.Handle(alice, "/pin abc")
	if !has(drain(alice), EvError, "6 digits") {
		t.Error("a bad PIN was accepted")
	}
}

func TestOnlyTheOwnerCanUseOwnerCommands(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	e.hub.Handle(alice, "/create-private secret 123456")
	e.hub.Handle(bob, "/join secret 123456")
	drain(alice)
	drain(bob)

	for _, input := range []string{"/pin 111111", "/remove alice", "/clear"}{
		e.hub.Handle(bob, input)
		if !has(drain(bob), EvError, "Only the owner") {
			t.Errorf("%q worked for a member who is not the owner", input)
		}
	}

	e.hub.Handle(bob, "/leave")
	drain(bob)
	e.hub.Handle(bob, "/pin 111111")
	if !has(drain(bob), EvError, "Only the owner") {
		t.Error("/pin worked in a public room")
	}
}


func TestOwnerRemovesAMember(t *testing.T) {
	e := newEnv(t)
	alice, bob, carol := e.connect("alice"), e.connect("bobby"), e.connect("carol")
	e.hub.Handle(alice, "/create-private secret 123456")
	e.hub.Handle(bob, "/join secret 123456")
	drain(alice)
	drain(bob)
	drain(carol)
	e.hub.Handle(alice, "/remove ghost")
	e.hub.Handle(alice, "/remove alice")
	e.hub.Handle(alice, "/remove carol")
	e.hub.Handle(alice, "/remove")
	evs := drain(alice)
	if !has(evs, EvError, "No such user") || !has(evs, EvError, "yourself") ||
		!has(evs, EvError, "not a member") || !has(evs, EvError, "Usage") {
		t.Errorf("error events = %+v", evs)
		}

	e.hub.Handle(alice, "/remove bobby")
	if !has(drain(alice), EvNotice, "bobby was removed") {
		t.Error("alice got no confirmation")
	}
	bobEvs := drain(bob)
	if !has(bobEvs, EvError, "removed from #secret") || bob.Room().Name != "lobby"{
		t.Errorf("bobby events = %+v, room = %q", bobEvs, bob.Room().Name)
	}

	e.hub.Handle(bob, "/join secret")
	if bob.Room().Name == "secret" {
		t.Error("a removed member came back without the PIN")
	}
	e.hub.Handle(bob, "/join secret 123456")
	if bob.Room().Name != "secret" {
		t.Error("a removed member could not come back with the PIN")
	}
}

func TestFullRoomBlocksPostingUntilCleared(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	e.hub.Handle(alice, "/create-private secret 123456")
	e.hub.Handle(bob, "/join secret 123456")
	if err := e.st.SetCap("secret", 10); err != nil {
		t.Fatal(err)
	}
	drain(alice)
	drain(bob)
	e.hub.Handle(alice, "hello")
	e.hub.Handle(alice, "world!")
	evs := drain(alice)
	if !has(evs, EvError, "room is full") || count(evs, EvMessage) != 1{
		t.Fatalf("alice events = %+v", evs)
	}
	drain(bob)

	e.hub.Handle(bob, "/clear")
	if !has(drain(bob), EvError, "Only the owner") {
		t.Error("a member cleared the room")
	}

	e.hub.Handle(alice, "/clear")
	bobEvs := drain(bob)
	if count(bobEvs, EvRoom) != 1 || !has(bobEvs, EvSystem, "cleared") {
		t.Errorf("bobby events after /clear = %+v", bobEvs)
	}
	room, _ := e.st.RoomByName("secret")
	if msgs,_ := e.st.RecentMessages(room.ID,10); len(msgs) != 0 {
		t.Errorf("%d messages left after /clear", len(msgs))
	}
	drain(alice)
	e.hub.Handle(alice, "world!")
	if count(drain(alice), EvMessage) != 1 {
		t.Error("posting still blocked after /clear")
	}
}

func TestRoomWithoutOwnerHasNoOwnerCommands(t *testing.T){
	e := newEnv(t)
	alice, bob := e.connect("alice"), e.connect("bobby")
	e.hub.Handle(alice, "/create-private secret 123456")
	e.hub.Handle(bob, "/join secret 123456")
	if err := e.st.DeleteUser("alice"); err != nil {
		t.Fatal(err)
	}
	drain(bob)

	for _, input := range []string{"/pin 111111", "/remove alice", "/clear"} {
		e.hub.Handle(bob, input)
		if !has(drain(bob), EvError, "Only the owner") {
			t.Errorf("%q worked in a room with no owner", input)
		}
	}
}

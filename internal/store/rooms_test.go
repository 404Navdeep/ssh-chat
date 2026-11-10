### internal/store/rooms_test.go
package store

import (
	"errors"
	"testing"
)

func TestCreatePublicRoom(t *testing.T) {
	st := openTest(t)
	u := mkUser(t, st, "alice")
	room, err := st.CreatePublicRoom(u.ID, "#Go-Chat")
	if err != nil{
		t.Fatal(err)
	}
	if room.Name != "go-chat" || room.Private || room.OwnerID != u.ID ||
		room.RetentionDays != DefaultPublicRetentionDays || room.CapBytes != 0 {
		t.Errorf("unexpected room: %+v", room)
	}
	if got, err := st.RoomByName("#GO-chat"); err != nil || got.ID != room.ID {
		t.Errorf("RoomByName = %+v, %v", got, err)
	}
	if _, err := st.CreatePublicRoom(u.ID, "GO-CHAT"); !errors.Is(err, ErrRoomTaken) {
		t.Errorf("duplicate room error = %v", err)
	}
	for _, bad := range []string{"", "a", "bad name", "x_y", "waytoolongroomnamethatexceeds"} {
		if _, err := st.CreatePublicRoom(u.ID, bad); !errors.Is(err, ErrBadRoomName) {
			t.Errorf("name %q error = %v", bad, err)
		}
	}
}

func TestListPublicRoomsPutsLobbyFirst(t *testing.T){
	st := openTest(t)
	u := mkUser(t, st, "alice")
	st.CreatePublicRoom(u.ID, "zeta")
	st.CreatePublicRoom(u.ID, "alpha")
	if _, err := st.CreatePrivateRoom(u.ID, "hidden", "123456"); err != nil {
		t.Fatal(err)
	}
	rooms, err := st.ListPublicRooms()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range rooms {
		names = append(names, r.Name)
	}
	want := []string{"lobby", "alpha", "zeta"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want{
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
	all,err := st.ListRooms()
	if err != nil || len(all) != 4 {
		t.Errorf("ListRooms = %d rooms, %v; want 4", len(all), err)
	}
}

func TestCreatePrivateRoom(t *testing.T) {

	st := openTest(t)
	u := mkUser(t, st, "alice")
	room, err := st.CreatePrivateRoom(u.ID, "secret", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if !room.Private || room.OwnerID != u.ID || room.CapBytes != DefaultPrivateCapBytes ||
		room.RetentionDays != DefaultPrivateRetentionDays {
		t.Errorf("unexpected room: %+v", room)
	}
	if room.pinHash == "" || room.pinHash == "123456"{
		t.Error("PIN must be stored hashed")
	}
	if _, err := st.CreatePrivateRoom(u.ID, "other", "12345"); !errors.Is(err, ErrBadPIN) {
		t.Errorf("short PIN error = %v", err)
	}
	if _, err := st.CreatePrivateRoom(u.ID, "other", "abcdef"); !errors.Is(err, ErrBadPIN) {
		t.Errorf("letter PIN error = %v", err)
	}
	if _, err := st.CreatePrivateRoom(u.ID, "lobby", "123456"); !errors.Is(err, ErrRoomTaken) {
		t.Errorf("name of a public room error = %v", err)
	}
	if _, err := st.CreatePrivateRoom(999, "other", "123456"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown owner error = %v", err)
	}
}

func TestPrivateRoomLimit(t *testing.T){
	st := openTest(t)
	u := mkUser(t, st, "alice")
	for _, name := range []string{"room-a", "room-b", "room-c"} {
		if _, err := st.CreatePrivateRoom(u.ID, name, "123456"); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	if _, err := st.CreatePrivateRoom(u.ID, "room-d", "123456"); !errors.Is(err, ErrRoomLimit) {
		t.Fatalf("fourth room error = %v, want ErrRoomLimit", err)
	}
	if err := st.SetMaxPrivateRooms("alice", 4); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreatePrivateRoom(u.ID, "room-d", "123456"); err != nil{
		t.Errorf("fourth room after raising the limit: %v", err)
	}
}

func TestDeleteRoom(t *testing.T) {
	st := openTest(t)
	u := mkUser(t, st, "alice")
	room, _ := st.CreatePublicRoom(u.ID, "temp")
	if _, err := st.AddMessage(room.ID, u, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteRoom("#temp"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RoomByName("temp"); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted room still found: %v", err)
	}
	var n int
	st.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE room_id = ?`, room.ID).Scan(&n)
	if n != 0{
		t.Errorf("%d messages left after deleting the room", n)
	}
	if err := st.DeleteRoom("lobby"); !errors.Is(err, ErrProtected) {
		t.Errorf("deleting the lobby error = %v", err)
	}
	if err := st.DeleteRoom("nothing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown room error = %v", err)
	}
}


func TestSetCapAndRetention(t *testing.T) {
	st := openTest(t)
	u := mkUser(t, st, "alice")
	st.CreatePrivateRoom(u.ID, "secret", "123456")
	if err := st.SetCap("secret", 2<<20); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRetention("secret", 90); err != nil{
		t.Fatal(err)
	}
	room, _ := st.RoomByName("secret")
	if room.CapBytes != 2<<20 || room.RetentionDays != 90 {
		t.Errorf("room = %+v", room)
	}
	if err := st.SetCap("lobby", 100); !errors.Is(err, ErrPublicRoom) {
		t.Errorf("cap on a public room error = %v", err)
	}
	if err := st.SetCap("nothing", 100); !errors.Is(err, ErrNotFound) {
		t.Errorf("cap on unknown room error = %v", err)
	}
	if err := st.SetRetention("lobby", 10); err != nil {
		t.Errorf("retention on a public room: %v", err)
	}
}

func TestRemovedOwnerLeavesRoomWithoutOwner(t *testing.T){
	st := openTest(t)
	u := mkUser(t, st, "alice")
	room, _ := st.CreatePrivateRoom(u.ID, "secret", "123456")
	if _, err := st.AddMessage(room.ID, u, "keep me"); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteUser("alice"); err != nil {
		t.Fatal(err)
	}
	got, err := st.RoomByName("secret")
	if err != nil || got.OwnerID != 0 {
		t.Fatalf("room after owner removal = %+v, %v", got, err)
	}
	msgs, err := st.RecentMessages(room.ID, 10)
	if err != nil || len(msgs) != 1 || msgs[0].Nick != "alice" || msgs[0].UserID != 0 {
		t.Errorf("messages after owner removal = %+v, %v", msgs, err)
	}
}

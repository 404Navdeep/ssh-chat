package store

import (
	"testing"
	"errors"
	"time"
)

func mkUser(t *testing.T, st *Store, nick string) User {
	t.Helper()
	u, err := st.CreateUser(nick, "fp-"+nick, "key-"+nick)
	if err != nil {
		t.Fatalf("CreateUser(%s): %v", nick, err)
	}
	return u
}

func TestCreateAndFindUser(t *testing.T) {
	st := openTest(t)
	u := mkUser(t, st, "alice")
	if u.ID == 0 || u.Nick != "alice" || u.Banned || u.MaxPrivateRooms != 3 {
		t.Errorf("unexpected user: %+v", u)
	}
	byKey, err := st.UserByFingerprint("fp-alice")
	if err != nil || byKey.ID != u.ID{
		t.Errorf("UserByFingerprint = %+v, %v", byKey, err)
	}
	byNick, err := st.UserByNick("ALICE")
	if err != nil || byNick.ID != u.ID {
		t.Errorf("UserByNick(ALICE) = %+v, %v", byNick, err)
	}
	if _, err := st.UserByFingerprint("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown key error = %v", err)
	}
}

func TestCreateUserRejectsBadInput(t *testing.T) {
	st := openTest(t)
	mkUser(t, st, "alice")
	if _, err := st.CreateUser("ab", "fp-1", "k"); !errors.Is(err, ErrBadNick) {
		t.Errorf("short nick error = %v", err)
	}
	if _, err := st.CreateUser("bad nick", "fp-2", "k"); !errors.Is(err, ErrBadNick){
		t.Errorf("space nick error = %v", err)
	}
	if _, err := st.CreateUser("ALICE", "fp-3", "k"); !errors.Is(err, ErrNickTaken) {
		t.Errorf("duplicate nick error = %v", err)
	}
	if _, err := st.CreateUser("bobby", "fp-alice", "k"); !errors.Is(err, ErrKeyTaken) {
		t.Errorf("duplicate key error = %v", err)
	}
}

func TestAddKey(t *testing.T) {
	st := openTest(t)
	u := mkUser(t, st, "alice")
	mkUser(t, st, "bobby")
	if err := st.AddKey("alice", "fp-second", "key-second"); err != nil {
		t.Fatal(err)
	}
	got, err := st.UserByFingerprint("fp-second")
	if err != nil || got.ID != u.ID{
		t.Errorf("second key resolves to %+v, %v", got, err)
	}
	if err := st.AddKey("alice", "fp-bobby", "k"); !errors.Is(err, ErrKeyTaken) {
		t.Errorf("key of another user error = %v", err)
	}
	if err := st.AddKey("nobody", "fp-x", "k"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user error = %v", err)
	}
}

func TestBanAndUnban(t *testing.T) {

	st := openTest(t)
	mkUser(t, st, "alice")
	if err := st.SetBanned("alice", true); err != nil {
		t.Fatal(err)
	}
	u, _ := st.UserByNick("alice")
	if !u.Banned{
		t.Error("alice should be banned")
	}
	if err := st.SetBanned("alice", false); err != nil {
		t.Fatal(err)
	}
	u, _ = st.UserByNick("alice")
	if u.Banned {
		t.Error("alice should be unbanned")
	}
	if err := st.SetBanned("ghost", true); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user error = %v", err)
	}
}
func TestSetMaxPrivateRooms(t *testing.T) {
	st := openTest(t)
	mkUser(t, st, "alice")
	if err := st.SetMaxPrivateRooms("alice", 7); err != nil{
		t.Fatal(err)
	}
	u, _ := st.UserByNick("alice")
	if u.MaxPrivateRooms != 7 {
		t.Errorf("MaxPrivateRooms = %d, want 7", u.MaxPrivateRooms)
	}
	if err := st.SetMaxPrivateRooms("ghost", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user error = %v", err)
	}
}

func TestDeleteUserFreesNickAndKeys(t *testing.T) {
	st := openTest(t)
	mkUser(t, st, "alice")
	mkUser(t, st, "bobby")
	if err := st.DeleteUser("alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UserByNick("alice"); !errors.Is(err, ErrNotFound){
		t.Errorf("deleted user still found: %v", err)
	}
	if _, err := st.UserByFingerprint("fp-alice"); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted key still found: %v", err)
	}
	if _, err := st.CreateUser("alice", "fp-new", "k"); err != nil {
		t.Errorf("nickname was not freed: %v", err)
	}
	if err := st.DeleteUser("ghost"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user error = %v", err)
	}
}

func TestListUsersIsSorted(t *testing.T) {
	st := openTest(t)
	mkUser(t, st, "carol")
	mkUser(t, st, "alice")
	mkUser(t, st, "bobby")
	users, err := st.ListUsers()
	if err != nil{
		t.Fatal(err)
	}
	if len(users) != 3 || users[0].Nick != "alice" || users[2].Nick != "carol" {
		t.Errorf("users = %+v", users)
	}
}

func TestRenameUserCooldown(t *testing.T) {
	st := openTest(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	st.Now = func() time.Time { return now }
	u := mkUser(t, st, "alice")
	mkUser(t, st, "bobby")

	renamed, err := st.RenameUser(u.ID, "alicia")
	if err != nil || renamed.Nick != "alicia" {
		t.Fatalf("first rename = %+v, %v", renamed, err)
	}
	_, err = st.RenameUser(u.ID, "alice2")
	var cool *CooldownError
	if !errors.As(err,&cool) {
		t.Fatalf("second rename error = %v, want CooldownError", err)
	}
	if !cool.Until.Equal(now.Add(NickCooldown)){
		t.Errorf("Until = %v", cool.Until)
	}

	now = now.Add(NickCooldown+time.Hour)
	if _, err := st.RenameUser(u.ID, "alice2"); err != nil {
		t.Errorf("rename after cooldown: %v", err)
	}
}

func TestRenameUserChecksNames(t *testing.T) {
	st := openTest(t)
	u := mkUser(t, st, "alice")
	mkUser(t, st, "bobby")
	if _, err := st.RenameUser(u.ID, "bobby"); !errors.Is(err, ErrNickTaken) {
		t.Errorf("taken nick error = %v", err)
	}
	if _, err := st.RenameUser(u.ID, "x"); !errors.Is(err, ErrBadNick) {
		t.Errorf("bad nick error = %v", err)
	}
	if _, err := st.RenameUser(u.ID, "alicia"); err != nil{
		t.Errorf("rename after failures: %v", err)
	}
}

package store

import (
	"testing"
	"errors"
	"time"
)

func secretRoom(t *testing.T, st *Store) Room{
	t.Helper()
	owner := mkUser(t, st, "owner")
	room, err := st.CreatePrivateRoom(owner.ID, "secret", "123456")
	if err != nil {
		t.Fatal(err)
	}
	return room
}
func TestOwnerIsMemberOfNewRoom(t *testing.T) {

	st := openTest(t)
	room := secretRoom(t, st)
	owner, _ := st.UserByNick("owner")
	if ok, err := st.IsMember(room.ID, owner.ID); err != nil || !ok {
		t.Errorf("owner IsMember = %v, %v", ok, err)
	}
}

func TestJoinPrivateWithRightPIN(t *testing.T) {
	st := openTest(t)
	room := secretRoom(t, st)
	bob := mkUser(t, st, "bobby")
	got, err := st.JoinPrivate(bob.ID, "#Secret", "123456")
	if err != nil || got.ID != room.ID{
		t.Fatalf("JoinPrivate = %+v, %v", got, err)
	}
	if ok, _ := st.IsMember(room.ID, bob.ID); !ok {
		t.Error("bobby should be a member after joining")
	}
}

func TestJoinPrivateHidesRoomNames(t *testing.T) {
	st := openTest(t)
	secretRoom(t, st)
	bob := mkUser(t, st, "bobby")
	if _, err := st.JoinPrivate(bob.ID, "secret", "000000"); !errors.Is(err, ErrWrongPIN) {
		t.Errorf("wrong PIN error = %v", err)
	}
	if _, err := st.JoinPrivate(bob.ID, "ghost", "123456"); !errors.Is(err, ErrWrongPIN) {
		t.Errorf("unknown room error = %v", err)
	}
	if _, err := st.JoinPrivate(bob.ID, "lobby", "123456"); !errors.Is(err, ErrWrongPIN){
		t.Errorf("public room error = %v", err)
	}
}


func TestPinLockout(t *testing.T) {
	st := openTest(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	st.Now = func() time.Time { return now }
	room := secretRoom(t,st)
	bob := mkUser(t, st, "bobby")
	carol := mkUser(t, st, "carol")

	for i := 1; i<=4; i++ {
		if _, err := st.JoinPrivate(bob.ID, "secret", "000000"); !errors.Is(err, ErrWrongPIN) {
			t.Fatalf("try %d error = %v, want ErrWrongPIN", i, err)
		}
	}
	_, err := st.JoinPrivate(bob.ID, "secret", "000000")
	var locked *LockedError
	if !errors.As(err, &locked) {
		t.Fatalf("fifth wrong try error = %v, want LockedError", err)
	}
	if !locked.Until.Equal(now.Add(PinLockout)){
		t.Errorf("Until = %v, want %v", locked.Until, now.Add(PinLockout))
	}

	if _, err := st.JoinPrivate(bob.ID, "secret", "123456"); !errors.As(err, &locked) {
		t.Errorf("right PIN while locked error = %v", err)
	}
	if _, err := st.JoinPrivate(carol.ID, "secret", "123456"); err != nil {
		t.Errorf("carol could not join: %v", err)
	}
	now = now.Add(PinLockout + time.Minute)
	got, err := st.JoinPrivate(bob.ID, "secret", "123456")
	if err != nil || got.ID != room.ID {
		t.Errorf("join after lockout = %+v, %v", got, err)
	}
}

func TestLockoutAlsoAppliesToUnknownRooms(t *testing.T) {
	st := openTest(t)
	bob := mkUser(t, st, "bobby")
	var err error
	for i := 0; i < MaxPinFails; i++{
		_, err = st.JoinPrivate(bob.ID, "ghost", "000000")
	}
	var locked *LockedError
	if !errors.As(err, &locked) {
		t.Errorf("error after %d tries = %v, want LockedError", MaxPinFails, err)
	}
}

func TestSuccessResetsFailureCount(t *testing.T) {
	st := openTest(t)
	secretRoom(t, st)
	bob := mkUser(t, st, "bobby")
	for i := 0; i<3; i++ {
		st.JoinPrivate(bob.ID, "secret", "000000")
	}
	if _, err := st.JoinPrivate(bob.ID, "secret", "123456"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++{
		if _, err := st.JoinPrivate(bob.ID, "secret", "000000"); !errors.Is(err, ErrWrongPIN) {
			t.Fatalf("try %d after success error = %v, want ErrWrongPIN", i+1, err)
		}
	}
}

func TestChangePIN(t *testing.T) {

	st := openTest(t)
	room := secretRoom(t, st)
	bob := mkUser(t, st, "bobby")
	carol := mkUser(t, st, "carol")
	if _, err := st.JoinPrivate(bob.ID, "secret", "123456"); err != nil {
		t.Fatal(err)
	}

	if err := st.ChangePIN(room.ID, "654321"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := st.IsMember(room.ID, bob.ID); !ok{
		t.Error("existing member must stay a member")
	}
	if _, err := st.JoinPrivate(carol.ID, "secret", "123456"); !errors.Is(err, ErrWrongPIN) {
		t.Errorf("old PIN error = %v", err)
	}
	if _, err := st.JoinPrivate(carol.ID, "secret", "654321"); err != nil {
		t.Errorf("new PIN: %v", err)
	}
	if err := st.ChangePIN(room.ID, "12"); !errors.Is(err, ErrBadPIN) {
		t.Errorf("bad PIN error = %v", err)
	}
	if err := st.ChangePIN(9999, "123456"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown room error = %v", err)
	}
}

func TestMembership(t *testing.T){
	st := openTest(t)
	room := secretRoom(t, st)
	bob := mkUser(t, st, "bobby")
	if ok,_ := st.IsMember(room.ID,bob.ID); ok {
		t.Fatal("bobby should not be a member yet")
	}
	if err := st.AddMember(room.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.AddMember(room.ID, bob.ID); err != nil {
		t.Errorf("adding twice: %v", err)
	}
	if err := st.RemoveMember(room.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if ok, _ := st.IsMember(room.ID, bob.ID); ok{
		t.Error("bobby should be removed")
	}
}

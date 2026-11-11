### internal/admin/admin_test.go
package admin

import (
	"crypto/ed25519"
	"bytes"
	"crypto/rand"
	"path/filepath"
	"strings"
	"testing"
	gossh "golang.org/x/crypto/ssh"

	"sshchat/internal/store"
)

func newKey(t *testing.T) string{
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(gossh.MarshalAuthorizedKey(sshPub)))
}

func openTest(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "admin.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func run(st *store.Store, args ...string) (string, error){
	var b bytes.Buffer
	err := Run(st, args, &b)

	return b.String(), err
}

func TestAddUserAndList(t *testing.T) {
	st := openTest(t)
	out, err := run(st, "adduser", "alice", newKey(t))
	if err != nil || !strings.Contains(out, "Added user alice") {
		t.Fatalf("adduser = %q, %v", out, err)
	}
	list, err := run(st, "listusers")
	if err != nil || !strings.Contains(list, "alice") || !strings.Contains(list, "active") {
		t.Errorf("listusers = %q, %v", list, err)
	}
}


func TestAddUserAcceptsKeySplitIntoWords(t *testing.T) {
	st := openTest(t)
	args := append([]string{"adduser", "alice"}, strings.Fields(newKey(t))...)
	if _, err := run(st, args...); err != nil{
		t.Errorf("adduser with split key: %v", err)
	}
}

func TestAddUserErrors(t *testing.T) {

	st := openTest(t)
	key := newKey(t)
	if _, err := run(st, "adduser", "alice", key); err != nil {
		t.Fatal(err)
	}
	if _, err := run(st, "adduser", "alice", newKey(t)); err == nil {
		t.Error("duplicate nickname should fail")
	}
	if _, err := run(st, "adduser", "bobby", key); err == nil {
		t.Error("duplicate key should fail")
	}
	if _, err := run(st, "adduser", "bobby", "not-a-key"); err == nil{
		t.Error("bad key should fail")
	}
	if _, err := run(st, "adduser", "x", newKey(t)); err == nil {
		t.Error("bad nickname should fail")
	}
	if _, err := run(st, "adduser", "alice"); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Errorf("missing key error = %v", err)
	}
}

func TestAddKeyGivesUserASecondKey(t *testing.T) {
	st := openTest(t)
	run(st, "adduser", "alice", newKey(t))
	second := newKey(t)
	if _, err := run(st, "addkey", "alice", second); err != nil {
		t.Fatal(err)
	}
	fp, _, _ := ParseKey(second)
	u,err := st.UserByFingerprint(fp)
	if err != nil || u.Nick != "alice"{
		t.Errorf("second key resolves to %+v, %v", u, err)
	}
	if _, err := run(st, "addkey", "ghost", newKey(t)); err == nil || !strings.Contains(err.Error(), "no user") {
		t.Errorf("unknown user error = %v", err)
	}
}
func TestBanUnbanRemove(t *testing.T) {
	st := openTest(t)
	run(st, "adduser", "alice", newKey(t))
	if _, err := run(st, "ban", "alice"); err != nil {
		t.Fatal(err)
	}
	if u, _ := st.UserByNick("alice"); !u.Banned {
		t.Error("alice should be banned")
	}
	if _, err := run(st, "unban", "alice"); err != nil{
		t.Fatal(err)
	}
	if u, _ := st.UserByNick("alice"); u.Banned {
		t.Error("alice should be unbanned")
	}
	if _, err := run(st, "removeuser", "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UserByNick("alice"); err == nil {
		t.Error("alice should be gone")
	}
	for _, cmd := range []string{"ban", "unban", "removeuser"} {
		if _, err := run(st, cmd, "ghost"); err == nil || !strings.Contains(err.Error(), "no user named ghost"){
			t.Errorf("%s ghost error = %v", cmd, err)
		}
	}
}

func TestRoomCommands(t *testing.T) {
	st := openTest(t)
	run(st, "adduser", "alice", newKey(t))
	u, _ := st.UserByNick("alice")
	if _, err := st.CreatePrivateRoom(u.ID, "secret", "123456"); err != nil {
		t.Fatal(err)
	}

	if _, err := run(st, "setcap", "#secret", "25"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(st, "setretention", "secret", "90"); err != nil {
		t.Fatal(err)
	}
	room, _ := st.RoomByName("secret")
	if room.CapBytes != 25<<20 || room.RetentionDays != 90{
		t.Errorf("room = %+v", room)
	}
	list, err := run(st, "listrooms")
	if err != nil || !strings.Contains(list, "#lobby") || !strings.Contains(list, "#secret") ||
		!strings.Contains(list, "25 MB") || !strings.Contains(list, "alice") {
		t.Errorf("listrooms = %q, %v", list, err)
	}

	if _, err := run(st, "setcap", "lobby", "5"); err == nil {
		t.Error("setcap on a public room should fail")
	}
	if _, err := run(st, "setcap", "secret", "0"); err == nil {
		t.Error("setcap 0 should fail")
	}
	if _, err := run(st, "setretention", "ghost", "5"); err == nil || !strings.Contains(err.Error(), "no room named #ghost") {
		t.Errorf("unknown room error = %v", err)
	}
	if _, err := run(st, "deleteroom", "lobby"); err == nil {
		t.Error("deleting the lobby should fail")
	}
	if _, err := run(st, "deleteroom", "secret"); err != nil{
		t.Fatal(err)
	}
	if _, err := st.RoomByName("secret"); err == nil {
		t.Error("secret should be deleted")
	}
}

func TestSetRoomLimit(t *testing.T) {
	st := openTest(t)
	run(st, "adduser", "alice", newKey(t))
	if _, err := run(st, "setroomlimit", "alice", "8"); err != nil {
		t.Fatal(err)
	}
	if u, _ := st.UserByNick("alice"); u.MaxPrivateRooms != 8 {
		t.Errorf("MaxPrivateRooms = %d, want 8", u.MaxPrivateRooms)
	}
	if _, err := run(st, "setroomlimit", "alice", "-1"); err == nil{
		t.Error("negative limit should fail")
	}
}


func TestUnknownAndMissingCommand(t *testing.T) {
	st := openTest(t)
	if _, err := run(st, "explode"); err == nil {
		t.Error("unknown command should fail")
	}
	if err := Run(st, nil, &bytes.Buffer{}); err == nil {
		t.Error("missing command should fail")
	}
	if !IsCommand("adduser") || IsCommand("explode") {
		t.Error("IsCommand gave a wrong answer")
	}
	if len(Usage()) != 11{
		t.Errorf("Usage has %d lines, want 11", len(Usage()))
	}
}

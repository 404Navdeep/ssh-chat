package store

import (
	"testing"
	"time"
)

func TestTakePendingDMs(t *testing.T) {
	st := openTest(t)
	alice := mkUser(t, st, "alice")
	bob := mkUser(t, st, "bobby")
	for _, body := range []string{"one", "two", "three"} {
		if err := st.AddDM("alice", bob.ID, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.AddDM("bobby", alice.ID, "for alice"); err != nil {
		t.Fatal(err)
	}

	first, err := st.TakePendingDMs(bob.ID, 2)
	if err != nil{
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Body != "one" || first[1].Body != "two" || first[0].FromNick != "alice" {
		t.Fatalf("first batch = %+v", first)
	}
	rest, err := st.TakePendingDMs(bob.ID, 10)
	if err != nil || len(rest) != 1 || rest[0].Body != "three" {
		t.Fatalf("second batch = %+v, %v", rest, err)
	}
	again,err := st.TakePendingDMs(bob.ID,10)
	if err != nil || len(again) != 0 {
		t.Errorf("third batch = %+v, %v; want empty", again, err)
	}
	forAlice, _ := st.TakePendingDMs(alice.ID, 10)
	if len(forAlice) != 1 || forAlice[0].Body != "for alice" {
		t.Errorf("alice's DMs = %+v", forAlice)
	}
}

func TestPurgeDMsAfter30Days(t *testing.T){
	st := openTest(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	st.Now = func() time.Time { return now }
	bob := mkUser(t, st, "bobby")
	if err := st.AddDM("alice", bob.ID, "old news"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(29 * 24 * time.Hour)
	if n, err := st.PurgeDMs(); err != nil || n != 0 {
		t.Fatalf("purge at 29 days = %d, %v; want 0", n, err)
	}
	now = now.Add(2*24*time.Hour)
	if n, err := st.PurgeDMs(); err != nil || n != 1 {
		t.Fatalf("purge at 31 days = %d, %v; want 1", n, err)
	}
	left, _ := st.TakePendingDMs(bob.ID, 10)
	if len(left) != 0 {
		t.Errorf("%d DMs left after purge", len(left))
	}
}

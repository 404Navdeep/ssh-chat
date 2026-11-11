### internal/chat/commands.go
package chat

import (
	"sort"
	"strings"
)

type cmdFunc func(h *Hub, s *Session, args string)

type command struct {
	usage string
	help string
	run cmdFunc
}

var commands = map[string]command{}

func register(name, usage, help string, run cmdFunc) {
	commands[name] = command{usage: usage, help: help, run: run}
}

func init() {
	register("help", "/help", "List the commands.", cmdHelp)
	register("quit", "/quit", "Leave the chat.", cmdQuit)
}
func (h *Hub) runCommand(s *Session, line string) {
	name, args, _ := strings.Cut(line[1:], " ")
	c, ok := commands[strings.ToLower(name)]
	if !ok{
		h.fail(s, "Unknown command. Type /help.")
		return
	}
	c.run(h, s, strings.TrimSpace(args))
}

func cmdHelp(h *Hub, s *Session, _ string) {
	names := make([]string,0,len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	h.say(s, "Commands:")
	for _, name := range names {
		c := commands[name]
		h.say(s, "  "+c.usage+"  "+c.help)
	}
}

func cmdQuit(h *Hub, s *Session, _ string) {

	h.tell(s, Event{Kind: EvClose, Body: "Goodbye.", At: h.Now()})
}

## Tests
### internal/chat/helpers_test.go
package chat

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sshchat/internal/store"
)

type clock struct{ t time.Time }

func (c *clock) Now() time.Time          { return c.t }
func (c *clock) Advance(d time.Duration) { c.t = c.t.Add(d) }

type env struct {
	t    *testing.T
	st   *store.Store
	hub  *Hub
	clk  *clock
	logs []string
}


func newEnv(t *testing.T) *env{
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	clk := &clock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	st.Now = clk.Now
	h := NewHub(st)
	h.Now = clk.Now
	e := &env{t: t, st: st, hub: h, clk: clk}
	h.Logf = func(format string, args ...interface{}) {
		e.logs = append(e.logs, fmt.Sprintf(format, args...))
	}
	return e
}

func (e *env) user(nick string) store.User {
	e.t.Helper()
	u, err := e.st.CreateUser(nick, "fp-"+nick, "key-"+nick)
	if err != nil {
		e.t.Fatalf("CreateUser(%s): %v", nick, err)
	}
	return u
}

func (e *env) connect(nick string) *Session {
	e.t.Helper()
	s,err := e.hub.Connect(e.user(nick))
	if err != nil{
		e.t.Fatalf("Connect(%s): %v", nick, err)
	}

	return s
}

func (e *env) logged(sub string) bool {
	for _, l := range e.logs {
		if strings.Contains(l, sub) { return true }
	}
	return false
}

func drain(s *Session) []Event {
	var out []Event
	for {
		select {
		case ev, ok := <-s.Events:
			if !ok{
				return out
			}
			out = append(out, ev)
		default:
			return out
		}
	}
}

func has(evs []Event, kind EventKind, sub string) bool {

	for _, ev := range evs {
		if ev.Kind==kind && strings.Contains(ev.Body, sub) { return true }
	}
	return false
}
func count(evs []Event, kind EventKind) int {
	n := 0
	for _,ev := range evs {

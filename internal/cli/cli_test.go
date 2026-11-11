### internal/cli/cli_test.go
package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func run(args ...string) (int, string, string) {

	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut, func(string) string { return "" })
	return code, out.String(), errOut.String()
}

func runner(t *testing.T) func(args ...string) (int, string, string) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "cli.db")
	return func(args ...string) (int, string, string) {
		var out, errOut bytes.Buffer
		getenv := func(k string) string {
			if k == "CHATD_DB" {
				return db
			}
			return ""
		}
		code := Run(args, &out, &errOut, getenv)
		return code,out.String(),errOut.String()
	}
}

func newKey(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil{
		t.Fatal(err)
	}
	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(gossh.MarshalAuthorizedKey(sshPub)))
}

func TestNoArgsPrintsUsage(t *testing.T) {
	code, out, _ := run()
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "chatd") {
		t.Errorf("usage text missing: %q", out)
	}
}

func TestHelpListsAdminCommands(t *testing.T){
	code, out, _ := run("help")
	if code != 0 || !strings.Contains(out, "CHATD_LISTEN") || !strings.Contains(out, "chatd adduser <nick> <pubkey>") {
		t.Errorf("code = %d, out = %q", code, out)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	code, _, errOut := run("bogus")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errOut, `unknown command "bogus"`) {
		t.Errorf("stderr = %q", errOut)
	}
}
func TestAdminCommandsShareOneDatabase(t *testing.T) {

	chatd := runner(t)
	if code, out, errOut := chatd("adduser", "alice", newKey(t)); code != 0{
		t.Fatalf("adduser: code %d, out %q, err %q", code, out, errOut)
	}
	code, out, _ := chatd("listusers")
	if code != 0 || !strings.Contains(out, "alice") {
		t.Errorf("listusers: code %d, out %q", code, out)
	}
	if code, _, errOut := chatd("ban", "ghost"); code != 1 || !strings.Contains(errOut, "no user named ghost") {
		t.Errorf("ban ghost: code %d, err %q", code, errOut)
	}
}

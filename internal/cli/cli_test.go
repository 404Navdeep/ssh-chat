package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut, func(string) string { return "" })
	return code, out.String(), errOut.String()
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

func TestHelpPrintsUsage(t *testing.T) {
	code, out, _ := run("help")
	if code != 0 || !strings.Contains(out, "CHATD_LISTEN") {
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

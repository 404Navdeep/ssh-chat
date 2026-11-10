// Package cli runs the chatd command line.
package cli

import (
	"fmt"
	"io"

	"sshchat/internal/config"
)

// Run runs one chatd command and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	cfg := config.FromEnv(getenv)
	if len(args) == 0 || isHelp(args[0]) {
		printUsage(stdout, cfg)
		return 0
	}
	fmt.Fprintf(stderr, "chatd: unknown command %q\n\n", args[0])
	printUsage(stderr, cfg)
	return 2
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func printUsage(w io.Writer, cfg config.Config) {
	fmt.Fprintln(w, "chatd - an SSH chat server")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  chatd help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Settings (environment variables):")
	fmt.Fprintf(w, "  CHATD_LISTEN    address to listen on (now: %s)\n", cfg.Listen)
	fmt.Fprintf(w, "  CHATD_HOST_KEY  host key file (now: %s)\n", cfg.HostKeyPath)
	fmt.Fprintf(w, "  CHATD_DB        database file (now: %s)\n", cfg.DBPath)
	fmt.Fprintf(w, "  CHATD_CONTACT   text for people without an account (now: %s)\n", cfg.Contact)
}

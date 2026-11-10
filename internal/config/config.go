// Package config holds the server settings, read from environment variables.
package config

import "strings"

// Default values, used when an environment variable is empty or missing.
const (
	DefaultListen      = ":2222"
	DefaultHostKeyPath = "chatd_host_key"
	DefaultDBPath      = "chatd.db"
	DefaultContact     = "Ask the server admin for an account."
)

// Config holds the server settings.
type Config struct {
	Listen      string // address the SSH server listens on
	HostKeyPath string // file that holds the server host key
	DBPath      string // SQLite database file
	Contact     string // text shown to people who have no account
}

// FromEnv reads the settings through getenv.
func FromEnv(getenv func(string) string) Config {
	return Config{
		Listen:      pick(getenv("CHATD_LISTEN"), DefaultListen),
		HostKeyPath: pick(getenv("CHATD_HOST_KEY"), DefaultHostKeyPath),
		DBPath:      pick(getenv("CHATD_DB"), DefaultDBPath),
		Contact:     pick(getenv("CHATD_CONTACT"), DefaultContact),
	}
}

func pick(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

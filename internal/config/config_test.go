package config

import "testing"

func TestFromEnvDefaults(t *testing.T) {
	cfg := FromEnv(func(string) string { return "" })
	if cfg.Listen != DefaultListen {
		t.Errorf("Listen = %q, want %q", cfg.Listen, DefaultListen)
	}
	if cfg.HostKeyPath != DefaultHostKeyPath {
		t.Errorf("HostKeyPath = %q, want %q", cfg.HostKeyPath, DefaultHostKeyPath)
	}
	if cfg.DBPath != DefaultDBPath {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, DefaultDBPath)
	}
	if cfg.Contact != DefaultContact {
		t.Errorf("Contact = %q, want %q", cfg.Contact, DefaultContact)
	}
}

func TestFromEnvOverrides(t *testing.T) {
	env := map[string]string{
		"CHATD_LISTEN":   "127.0.0.1:3000",
		"CHATD_HOST_KEY": "/tmp/host_key",
		"CHATD_DB":       "/tmp/chat.db",
		"CHATD_CONTACT":  "Mail admin@example.com",
	}
	cfg := FromEnv(func(k string) string { return env[k] })
	if cfg.Listen != "127.0.0.1:3000" {
		t.Errorf("Listen = %q", cfg.Listen)
	}
	if cfg.HostKeyPath != "/tmp/host_key" {
		t.Errorf("HostKeyPath = %q", cfg.HostKeyPath)
	}
	if cfg.DBPath != "/tmp/chat.db" {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if cfg.Contact != "Mail admin@example.com" {
		t.Errorf("Contact = %q", cfg.Contact)
	}
}

func TestBlankValueFallsBack(t *testing.T) {
	cfg := FromEnv(func(k string) string { return "   " })
	if cfg.Listen != DefaultListen {
		t.Errorf("Listen = %q, want default", cfg.Listen)
	}
}

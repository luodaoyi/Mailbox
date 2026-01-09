package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`server:
  http_port: 8080
  smtp_port: 25

database:
  host: localhost
  port: 3306
  user: root
  password: pass
  name: mailbox

admin:
  default_user: admin
  default_pass: admin123
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "mailbox_test")
	t.Setenv("ADMIN_USERNAME", "root")
	t.Setenv("ADMIN_PASSWORD", "rootpass")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.Database.Host != "db" || cfg.Database.Port != 3307 || cfg.Database.User != "user" || cfg.Database.Password != "secret" || cfg.Database.Name != "mailbox_test" {
		t.Fatalf("db override not applied")
	}
	if cfg.Admin.DefaultUser != "root" || cfg.Admin.DefaultPass != "rootpass" {
		t.Fatalf("admin override not applied")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("not-exist.yaml")
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}

package main

import (
	"embed"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"mailbox/internal/config"
)

func TestMain(m *testing.M) {
	origStart := startSMTP
	origLoad := loadEnv
	startSMTP = func(_ int) error { return nil }
	loadEnv = func(_ ...string) error { return nil }
	code := m.Run()
	startSMTP = origStart
	loadEnv = origLoad
	os.Exit(code)
}

func writeTestConfig(t *testing.T, dir string) string {
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
	return path
}

func TestRunMissingJWT(t *testing.T) {
	cfgPath := writeTestConfig(t, t.TempDir())
	origInit := initDB
	origStartSMTP := startSMTP
	origSetupApp := setupApp
	origRun := runFunc
	origLoadEnv := loadEnv
	t.Cleanup(func() {
		initDB = origInit
		startSMTP = origStartSMTP
		setupApp = origSetupApp
		runFunc = origRun
		loadEnv = origLoadEnv
	})

	initDB = func(_ *config.DatabaseConfig, _ *config.AdminConfig) error { return nil }
	startSMTP = func(_ int) error { return nil }
	setupApp = func(_ embed.FS, _ string) *fiber.App { return fiber.New() }
	loadEnv = func(_ ...string) error { return nil }

	os.Unsetenv("JWT_SECRET")
	if err := run(cfgPath, embed.FS{}, func(_ *fiber.App, _ int) error { return nil }); err == nil {
		t.Fatalf("expected jwt error")
	}
}

func TestRunSuccess(t *testing.T) {
	cfgPath := writeTestConfig(t, t.TempDir())
	origInit := initDB
	origStartSMTP := startSMTP
	origSetupApp := setupApp
	origFatalf := fatalf
	origRun := runFunc
	origLoadEnv := loadEnv
	t.Cleanup(func() {
		initDB = origInit
		startSMTP = origStartSMTP
		setupApp = origSetupApp
		fatalf = origFatalf
		runFunc = origRun
		loadEnv = origLoadEnv
	})

	initDB = func(_ *config.DatabaseConfig, _ *config.AdminConfig) error { return nil }
	setupApp = func(_ embed.FS, _ string) *fiber.App { return fiber.New() }
	loadEnv = func(_ ...string) error { return nil }
	startSMTPCh := make(chan int, 1)
	startSMTP = func(port int) error {
		startSMTPCh <- port
		return nil
	}
	fatalf = func(format string, args ...any) {
		t.Fatalf("fatal called")
	}

	os.Setenv("JWT_SECRET", "secret")
	if err := run(cfgPath, embed.FS{}, func(_ *fiber.App, port int) error {
		if port != 8080 {
			t.Fatalf("unexpected http port: %d", port)
		}
		return nil
	}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	select {
	case port := <-startSMTPCh:
		if port != 25 {
			t.Fatalf("unexpected smtp port: %d", port)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("smtp not started")
	}
}

func TestMainUsesRunFunc(t *testing.T) {
	origRun := runFunc
	origLoadEnv := loadEnv
	runCalled := false
	runFunc = func(_ string, _ embed.FS, _ func(app *fiber.App, port int) error) error {
		runCalled = true
		return nil
	}
	loadEnv = func(_ ...string) error { return nil }
	t.Cleanup(func() {
		runFunc = origRun
		loadEnv = origLoadEnv
	})

	main()
	if !runCalled {
		t.Fatalf("run not called")
	}
}

func TestRunConfigError(t *testing.T) {
	origInit := initDB
	origStartSMTP := startSMTP
	origSetupApp := setupApp
	origFatalf := fatalf
	origLoadEnv := loadEnv
	t.Cleanup(func() {
		initDB = origInit
		startSMTP = origStartSMTP
		setupApp = origSetupApp
		fatalf = origFatalf
		loadEnv = origLoadEnv
	})

	initDB = func(_ *config.DatabaseConfig, _ *config.AdminConfig) error { return nil }
	startSMTP = func(_ int) error { return nil }
	setupApp = func(_ embed.FS, _ string) *fiber.App { return fiber.New() }
	fatalf = func(string, ...any) {}
	loadEnv = func(_ ...string) error { return nil }

	os.Setenv("JWT_SECRET", "secret")
	if err := run("not-exist.yaml", embed.FS{}, func(_ *fiber.App, _ int) error { return nil }); err == nil {
		t.Fatalf("expected config error")
	}
}

func TestRunInitDBError(t *testing.T) {
	cfgPath := writeTestConfig(t, t.TempDir())
	origInit := initDB
	origStartSMTP := startSMTP
	origSetupApp := setupApp
	origFatalf := fatalf
	origLoadEnv := loadEnv
	t.Cleanup(func() {
		initDB = origInit
		startSMTP = origStartSMTP
		setupApp = origSetupApp
		fatalf = origFatalf
		loadEnv = origLoadEnv
	})

	initDB = func(_ *config.DatabaseConfig, _ *config.AdminConfig) error { return errors.New("db failed") }
	startSMTP = func(_ int) error { return nil }
	setupApp = func(_ embed.FS, _ string) *fiber.App { return fiber.New() }
	fatalf = func(string, ...any) {}
	loadEnv = func(_ ...string) error { return nil }

	os.Setenv("JWT_SECRET", "secret")
	if err := run(cfgPath, embed.FS{}, func(_ *fiber.App, _ int) error { return nil }); err == nil {
		t.Fatalf("expected init error")
	}
}

func TestRunStartHTTPError(t *testing.T) {
	cfgPath := writeTestConfig(t, t.TempDir())
	origInit := initDB
	origStartSMTP := startSMTP
	origSetupApp := setupApp
	origFatalf := fatalf
	origLoadEnv := loadEnv
	t.Cleanup(func() {
		initDB = origInit
		startSMTP = origStartSMTP
		setupApp = origSetupApp
		fatalf = origFatalf
		loadEnv = origLoadEnv
	})

	initDB = func(_ *config.DatabaseConfig, _ *config.AdminConfig) error { return nil }
	startSMTP = func(_ int) error { return nil }
	setupApp = func(_ embed.FS, _ string) *fiber.App { return fiber.New() }
	fatalf = func(string, ...any) {}
	loadEnv = func(_ ...string) error { return nil }

	os.Setenv("JWT_SECRET", "secret")
	if err := run(cfgPath, embed.FS{}, func(_ *fiber.App, _ int) error { return errors.New("http failed") }); err == nil {
		t.Fatalf("expected http error")
	}
}

func TestRunStartSMTPError(t *testing.T) {
	cfgPath := writeTestConfig(t, t.TempDir())
	origInit := initDB
	origStartSMTP := startSMTP
	origSetupApp := setupApp
	origFatalf := fatalf
	origLoadEnv := loadEnv
	t.Cleanup(func() {
		initDB = origInit
		startSMTP = origStartSMTP
		setupApp = origSetupApp
		fatalf = origFatalf
		loadEnv = origLoadEnv
	})

	initDB = func(_ *config.DatabaseConfig, _ *config.AdminConfig) error { return nil }
	setupApp = func(_ embed.FS, _ string) *fiber.App { return fiber.New() }

	fatalCh := make(chan struct{}, 1)
	startSMTP = func(_ int) error { return errors.New("smtp failed") }
	fatalf = func(string, ...any) { fatalCh <- struct{}{} }
	loadEnv = func(_ ...string) error { return nil }

	os.Setenv("JWT_SECRET", "secret")
	if err := run(cfgPath, embed.FS{}, func(_ *fiber.App, _ int) error { return nil }); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	select {
	case <-fatalCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("smtp fatal not called")
	}
}

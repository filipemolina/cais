package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigHome points XDG_CONFIG_HOME at dir for the test's lifetime,
// so LoadConfig/SaveConfig read from there instead of the real home.
func withConfigHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
}

func TestLoadConfigMissingFile(t *testing.T) {
	withConfigHome(t, t.TempDir())

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("missing config file should not error: %v", err)
	}
	if cfg.Theme != "" {
		t.Errorf("missing config should yield empty theme, got %q", cfg.Theme)
	}
}

func TestRoundTrip(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	original := Config{Theme: "cais-day"}
	if err := SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Theme != original.Theme {
		t.Errorf("round-tripped theme = %q, want %q", loaded.Theme, original.Theme)
	}

	// The file should live at $XDG_CONFIG_HOME/cais/config.yaml.
	path := filepath.Join(home, "cais", "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not found at %s: %v", path, err)
	}
}

func TestLoadConfigMalformed(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	dir := filepath.Join(home, "cais")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("[invalid"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("malformed config should return an error")
	}
}

func TestSaveConfigCreatesDirectory(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	dir := filepath.Join(home, "cais")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected %s to not exist yet", dir)
	}

	if err := SaveConfig(Config{Theme: "cais-dark"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("SaveConfig should have created %s: %v", dir, err)
	}
}

func TestSaveConfigOverwrites(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	if err := SaveConfig(Config{Theme: "cais-dusk"}); err != nil {
		t.Fatalf("first SaveConfig: %v", err)
	}
	if err := SaveConfig(Config{Theme: "cais-day"}); err != nil {
		t.Fatalf("second SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Theme != "cais-day" {
		t.Errorf("overwritten theme = %q, want %q", loaded.Theme, "cais-day")
	}
}

// writeLegacyConfig seeds an old-identity stack-stitcher directory with a
// config.yaml fixture that the migration tests parametrize with `themeValue`.
func writeLegacyConfig(t *testing.T, home, themeValue string) string {
	t.Helper()
	legacy := filepath.Join(home, "stack-stitcher")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if themeValue != "" {
		body := "theme: " + themeValue + "\n"
		if err := os.WriteFile(filepath.Join(legacy, "config.yaml"), []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return legacy
}

func TestMigrateLegacyConfigNoLegacyDir(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	res, err := MigrateLegacyConfig()
	if err != nil {
		t.Fatalf("MigrateLegacyConfig on empty home: %v", err)
	}
	if res.Migrated {
		t.Error("Migrated should be false when no legacy dir exists")
	}
	if res.ThemeRenamed {
		t.Error("ThemeRenamed should be false when nothing was rewritten")
	}
	if res.Warning != "" {
		t.Errorf("Warning = %q, want empty", res.Warning)
	}
}

func TestMigrateLegacyConfigBothDirsPresent(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	legacy := writeLegacyConfig(t, home, "stitcher-dark")
	newDir := filepath.Join(home, "cais")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("MkdirAll new: %v", err)
	}

	res, err := MigrateLegacyConfig()
	if err != nil {
		t.Fatalf("MigrateLegacyConfig with both dirs: %v", err)
	}
	if res.Migrated {
		t.Error("Migrated should be false when both dirs exist")
	}
	if res.Warning == "" {
		t.Error("Warning should be non-empty when both dirs exist")
	}
	// The legacy dir must still exist; the migration refuses to touch
	// either directory in this state.
	if _, err := os.Stat(legacy); err != nil {
		t.Errorf("legacy dir %s should still exist: %v", legacy, err)
	}
}

func TestMigrateLegacyConfigRenamesDirAndRewritesTheme(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	legacy := writeLegacyConfig(t, home, "stitcher-ember")
	newDir := filepath.Join(home, "cais")

	res, err := MigrateLegacyConfig()
	if err != nil {
		t.Fatalf("MigrateLegacyConfig: %v", err)
	}
	if !res.Migrated {
		t.Error("Migrated should be true when legacy dir exists and new does not")
	}
	if res.From != legacy {
		t.Errorf("From = %q, want %q", res.From, legacy)
	}
	if res.To != newDir {
		t.Errorf("To = %q, want %q", res.To, newDir)
	}
	if !res.ThemeRenamed {
		t.Error("ThemeRenamed should be true when stitcher-ember is rewritten to a cais-* theme")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy dir should be gone, stat err = %v", err)
	}
	if _, err := os.Stat(newDir); err != nil {
		t.Errorf("new dir should exist: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	// stitcher-ember maps to cais-dusk in legacyThemeMap.
	if cfg.Theme != "cais-dusk" {
		t.Errorf("loaded theme = %q, want %q", cfg.Theme, "cais-dusk")
	}
}

func TestMigrateLegacyConfigAlreadyCaisThemeUntouched(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	writeLegacyConfig(t, home, "cais-dark")

	res, err := MigrateLegacyConfig()
	if err != nil {
		t.Fatalf("MigrateLegacyConfig: %v", err)
	}
	if !res.Migrated {
		t.Error("Migrated should be true on a legacy dir with a cais-* theme")
	}
	if res.ThemeRenamed {
		t.Error("ThemeRenamed should be false when the saved theme was already cais-*")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Theme != "cais-dark" {
		t.Errorf("loaded theme = %q, want %q (no rewrite)", cfg.Theme, "cais-dark")
	}
}

func TestMigrateLegacyConfigIsIdempotent(t *testing.T) {
	home := t.TempDir()
	withConfigHome(t, home)

	writeLegacyConfig(t, home, "stitcher-day")

	// First migration: rename + theme rewrite.
	if _, err := MigrateLegacyConfig(); err != nil {
		t.Fatalf("first MigrateLegacyConfig: %v", err)
	}
	// Second migration: legacy dir is gone now, should be a silent no-op.
	res, err := MigrateLegacyConfig()
	if err != nil {
		t.Fatalf("second MigrateLegacyConfig: %v", err)
	}
	if res.Migrated {
		t.Error("Migrated should be false on the second call (legacy already moved)")
	}
	if res.ThemeRenamed {
		t.Error("ThemeRenamed should be false on the second call")
	}
}

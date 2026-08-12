// Package config reads and writes the user's persistent preferences:
// ~/.config/cais/config.yaml (or $XDG_CONFIG_HOME if set).
//
// The config file is a small YAML document:
//
//	theme: cais-dark
//
// More fields will land later (default file, keybinding overrides — see
// docs/ROADMAP.md). The struct and the write path are designed to absorb
// them without changing existing callers: Add a field, tag it, and
// LoadConfig/SaveConfig round-trip it automatically.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the persistent user preferences. Fields are exported for YAML
// marshalling; the zero value of each is the "not set" sentinel, and
// LoadConfig leaves a missing field at its zero rather than erroring.
type Config struct {
	// Theme is the registered theme name to activate on startup.
	// Empty means "use appstyles.DefaultTheme".
	Theme string `yaml:"theme,omitempty"`

	// URLHost overrides the host part of every service URL the app builds
	// (utils.URLHost). Empty means "detect it" - SSH_CONNECTION's server
	// address when running over SSH, "localhost" otherwise.
	URLHost string `yaml:"url_host,omitempty"`
}

// appDir is the directory name the app's state lives under. Renamed
// from "stack-stitcher" when the project rebranded. Migration of an
// existing "stack-stitcher" directory runs once at first launch —
// see MigrateLegacyConfig.
const appDir = "cais"

// legacyAppDir is the directory name the previous identity used. Kept
// here so a future appstylevar rename can re-purpose the same one-shot
// migration without re-engineering the search.
const legacyAppDir = "stack-stitcher"

// legacyThemeMap maps each theme name from the previous identity's
// palette to its closest Cais replacement, so a migrated config.yaml's
// `theme:` value keeps a sane-looking theme on launch. Unknown names
// are left untouched — SetTheme falls back to DefaultTheme for those.
var legacyThemeMap = map[string]string{
	"stitcher-dark":  "cais-dark",
	"stitcher-ember": "cais-dusk",
	"stitcher-slate": "cais-dusk",
	"stitcher-day":   "cais-day",
}

// MigrateResult reports what MigrateLegacyConfig did on its one-shot run.
// The CLI/TUI surfaces the migrated message once on its first run.
type MigrateResult struct {
	// Migrated is true if the legacy directory was found and renamed into
	// place under the new appDir. False means there was nothing to do.
	Migrated bool
	// From and To are the directories involved in the migration. From is
	// empty when the legacy directory did not exist on startup.
	From, To string
	// ThemeRenamed is true if a stitcher-* theme in config.yaml was
	// rewritten to a cais-* equivalent during the migration.
	ThemeRenamed bool
	// Warning is non-empty when both legacy and new directories exist on
	// startup. Neither is touched in that case — the user has to decide.
	Warning string
}

// configDir returns the directory the config file lives in:
// $XDG_CONFIG_HOME/cais if XDG_CONFIG_HOME is set, otherwise
// ~/.config/cais.
func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appDir), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", appDir), nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// legacyConfigDir returns where the previous-identity config dir would
// live (parallel layout to configDir, with the legacy slug). Used only
// by MigrateLegacyConfig.
func legacyConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, legacyAppDir), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", legacyAppDir), nil
}

// MigrateLegacyConfig performs a one-shot, first-launch migration from
// the previous-identity "stack-stitcher" config directory to "cais".
//
// The rules:
//   - legacy dir absent → no-op (first-time install).
//   - new dir already exists → no-op + warning. The user has both
//     directories; Cais must not delete or overwrite either.
//   - legacy dir present, new dir absent → rename legacy→new; rewrite
//     any `theme: stitcher-*` line in config.yaml to its cais-*
//     equivalent. The DB file (if any) is renamed along with the dir.
//
// The result struct reports what happened so a caller can surface a
// one-time notice. MigrateLegacyConfig is safe to call repeatedly; once
// the legacy dir is gone, it returns Migrated=false silently.
func MigrateLegacyConfig() (MigrateResult, error) {
	_, err := configDir()
	if err != nil {
		return MigrateResult{}, err
	}
	newDir, _ := configDir()

	legacyDir, err := legacyConfigDir()
	if err != nil {
		return MigrateResult{}, err
	}

	// legacy absent: nothing to do — clean first-time install path.
	if _, err := os.Stat(legacyDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MigrateResult{}, nil
		}
		return MigrateResult{}, err
	}

	// both exist: refuse to migrate. The user owns both directories and
	// must choose which one wins; we surface this rather than deleting.
	if _, err := os.Stat(newDir); err == nil {
		return MigrateResult{
			Migrated: false,
			From:     legacyDir,
			To:       newDir,
			Warning:  "both " + legacyAppDir + " and " + appDir + " exist on disk; not migrating",
		}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return MigrateResult{}, err
	}

	if err := os.Rename(legacyDir, newDir); err != nil {
		return MigrateResult{}, err
	}

	res := MigrateResult{
		Migrated: true,
		From:     legacyDir,
		To:       newDir,
	}

	// Rewrite the persisted theme name in config.yaml, if any. We do this
	// after the rename so a crash mid-rewrite still leaves the directory
	// in the right place; the only loss is one re-applied theme.
	cfgFile := filepath.Join(newDir, "config.yaml")
	data, err := os.ReadFile(cfgFile)
	if err == nil {
		rewritten, changed := rewriteLegacyTheme(string(data))
		if changed {
			if writeErr := os.WriteFile(cfgFile, []byte(rewritten), 0o644); writeErr != nil {
				return res, writeErr
			}
			res.ThemeRenamed = true
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return res, err
	}

	return res, nil
}

// rewriteLegacyTheme rewrites any `theme: stitcher-*` value in a
// config.yaml text to its cais-* equivalent. Returns the rewritten
// text plus a flag saying whether anything actually changed.
//
// The shape it scans for is `theme: <name>` on a line of its own; the
// YAML this app writes keeps theme on its own line, so this is good
// enough. (A more thorough path would re-parse + re-marshal the file,
// but that's a bigger swing than this one-shot needs.)
func rewriteLegacyTheme(s string) (string, bool) {
	changed := false
	out := strings.Builder{}
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "theme:") {
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "theme:"))
			// Skip comments / blank values.
			if rest == "" || strings.HasPrefix(rest, "#") {
				out.WriteString(line)
				out.WriteByte('\n')
				continue
			}
			if mapped, ok := legacyThemeMap[rest]; ok {
				// Preserve the original leading whitespace + the "theme:"
				// prefix exactly so the only diff in the file is the
				// theme value itself.
				prefix := line[:len(line)-len(strings.TrimLeft(line, " 	"))]
				out.WriteString(prefix)
				out.WriteString("theme: ")
				out.WriteString(mapped)
				out.WriteByte('\n')
				changed = true
				continue
			}
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String(), changed
}

// LoadConfig reads the config file and returns the parsed Config. A missing
// file is not an error: it returns the zero Config and a nil error, so the
// caller can apply defaults (DefaultTheme) without special-casing "first run".
// A malformed file is an error worth reporting — the caller decides whether
// to surface it or fall back to defaults.
func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// SaveConfig writes cfg to the config file, creating the directory if
// needed. It is the whole persistence story for now: one call after a
// theme is chosen, one file, one write.
func SaveConfig(cfg Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0o644)
}

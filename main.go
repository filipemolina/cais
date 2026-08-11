package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/config"
	"github.com/filipemolina/cais/src/constants"
	"github.com/filipemolina/cais/src/model"
	"github.com/filipemolina/cais/src/utils"
)

// errVersionRequested is how parseFlags reports --version. It is not a
// failure, and it is not a ComposeSource either: the run ends after printing.
var errVersionRequested = errors.New("version requested")

func main() {
	source, err := parseFlags(os.Args[1:])
	if errors.Is(err, errVersionRequested) {
		fmt.Println("cais", constants.Version())
		os.Exit(0)
	}
	if err != nil {
		// -h/--help has already printed the usage text, and asking for help
		// is not a failure.
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "cais:", err)
		os.Exit(1)
	}

	// One-shot migration from the previous-identity config dir. Runs
	// before anything reads config so LoadConfig sees the new layout.
	// Errors are non-fatal: a fresh install may have no legacy dir,
	// and a partial legacy needs the user to look, not the app to
	// refuse to start.
	if res, err := config.MigrateLegacyConfig(); err == nil {
		if res.Warning != "" {
			fmt.Fprintln(os.Stderr, "cais:", res.Warning)
		} else if res.Migrated && res.ThemeRenamed {
			fmt.Fprintln(os.Stderr, "cais: migrated", res.From, "→", res.To, "(theme renamed to cais-*)")
		} else if res.Migrated {
			fmt.Fprintln(os.Stderr, "cais: migrated", res.From, "→", res.To)
		}
	}

	// Apply the saved theme before the program starts. A missing or
	// malformed config is silently ignored — the default theme is
	// already active. An unknown theme name is also ignored: the
	// registry may have shrunk since the config was written.
	if cfg, err := config.LoadConfig(); err == nil && cfg.Theme != "" {
		appstyles.SetTheme(cfg.Theme)
	}

	p := tea.NewProgram(model.GetInitialModel(source))

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

// parseFlags turns the command line into the one question the app needs
// answered before it starts: which compose file to work on.
//
// Both problems it can report - a directory that isn't one, a file that isn't
// there - are caught here rather than inside the app. A bad path is a mistake
// in the command that was just typed, and the honest place to say so is the
// shell it was typed into, not an error banner behind a TUI.
func parseFlags(args []string) (utils.ComposeSource, error) {
	var source utils.ComposeSource

	flags := flag.NewFlagSet("cais", flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), usage)
	}

	// Each flag is registered twice so the short and long spellings write to
	// the same variable; Go's flag package has no built-in aliases.
	flags.StringVar(&source.File, "file", "", "path to the compose file to open")
	flags.StringVar(&source.File, "f", "", "shorthand for --file")
	flags.StringVar(&source.Dir, "dir", "", "directory to look for a compose file in")
	flags.StringVar(&source.Dir, "d", "", "shorthand for --dir")

	var showVersion bool
	flags.BoolVar(&showVersion, "version", false, "print the version and exit")
	flags.BoolVar(&showVersion, "v", false, "shorthand for --version")

	if err := flags.Parse(args); err != nil {
		return utils.ComposeSource{}, err
	}

	if showVersion {
		return utils.ComposeSource{}, errVersionRequested
	}

	// --file names the file outright and --dir says where to go looking for
	// one; honouring both would mean picking which of two answers the user
	// meant. Either flag on its own already covers the other's use.
	if source.File != "" && source.Dir != "" {
		return utils.ComposeSource{}, fmt.Errorf("--file and --dir cannot be used together")
	}

	if source.Dir != "" {
		info, err := os.Stat(source.Dir)
		if err != nil {
			return utils.ComposeSource{}, fmt.Errorf("--dir %s: %w", source.Dir, err)
		}
		if !info.IsDir() {
			return utils.ComposeSource{}, fmt.Errorf("--dir %s is not a directory", source.Dir)
		}
	}

	if source.File != "" {
		info, err := os.Stat(source.File)
		if err != nil {
			return utils.ComposeSource{}, fmt.Errorf("--file %s: %w", source.File, err)
		}
		if info.IsDir() {
			return utils.ComposeSource{}, fmt.Errorf("--file %s is a directory (did you mean --dir?)", source.File)
		}
	}

	return source, nil
}

const usage = `Cais - a terminal UI for Docker Compose.

Usage:
  cais [flags]  

Flags:
  -f, --file PATH   open this compose file
  -d, --dir  PATH   look for a compose file in this directory
  -v, --version     print the version and exit
  -h, --help        show this help

With no flags, the compose file is resolved in the current directory, in
Docker's own order: compose.yaml, compose.yml, docker-compose.yaml,
docker-compose.yml.
`

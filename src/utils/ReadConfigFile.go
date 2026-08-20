package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

func ReadConfigFile(fileName string) (*types.Project, error) {
	projectName := "cais"
	ctx := context.Background()
	workingDir, wdErr := os.Getwd()
	if wdErr != nil {
		return nil, fmt.Errorf("failed reading working directory: %w", wdErr)
	}

	// Callers normally pass a bare name found in the working directory, but
	// an absolute path must be used as given - joining it onto the working
	// directory would produce a path that doesn't exist.
	path := fileName
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, fileName)
	}
	options, projectErr := cli.NewProjectOptions(
		[]string{path},
		cli.WithOsEnv,
		cli.WithEnvFiles(),
		cli.WithDotEnv,
		cli.WithName(projectName),
	)
	if projectErr != nil {
		return nil, fmt.Errorf("failed reading compose file options: %w", projectErr)
	}

	project, loadErr := options.LoadProject(ctx)
	if loadErr != nil {
		return nil, fmt.Errorf("failed loading compose file %s: %w", fileName, loadErr)
	}

	return project, nil
}

// ReadConfigFileExt returns the project, the resolved .env path, and whether it was loaded.
// The .env path is resolved relative to the compose file's directory (compose-go semantics).
// It may exist but not be loaded if COMPOSE_DISABLE_ENV_FILE is set.
func ReadConfigFileExt(fileName string) (*types.Project, string, bool, error) {
	projectName := "cais"
	ctx := context.Background()
	workingDir, wdErr := os.Getwd()
	if wdErr != nil {
		return nil, "", false, fmt.Errorf("failed reading working directory: %w", wdErr)
	}

	path := fileName
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, fileName)
	}

	// Resolve the .env path: compose-go looks for it in the directory of the
	// compose file (cli/options.go:410).
	composeDir := filepath.Dir(path)
	envPath := filepath.Join(composeDir, ".env")

	options, projectErr := cli.NewProjectOptions(
		[]string{path},
		cli.WithOsEnv,
		cli.WithEnvFiles(),
		cli.WithDotEnv,
		cli.WithName(projectName),
	)
	if projectErr != nil {
		return nil, envPath, false, fmt.Errorf("failed reading compose file options: %w", projectErr)
	}

	project, loadErr := options.LoadProject(ctx)
	if loadErr != nil {
		return nil, envPath, false, fmt.Errorf("failed loading compose file %s: %w", fileName, loadErr)
	}

	// Check if the .env was loaded by looking at options.EnvFiles
	envLoaded := false
	for _, file := range options.EnvFiles {
		if file == envPath {
			envLoaded = true
			break
		}
	}

	return project, envPath, envLoaded, nil
}

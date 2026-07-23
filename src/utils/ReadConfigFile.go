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
	projectName := "stack-stitcher"
	ctx := context.Background()
	workingDir, wdErr := os.Getwd()
	if wdErr != nil {
		return nil, fmt.Errorf("failed reading working directory: %w", wdErr)
	}

	path := filepath.Join(workingDir, fileName)
	options, projectErr := cli.NewProjectOptions(
		[]string{path},
		cli.WithOsEnv,
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

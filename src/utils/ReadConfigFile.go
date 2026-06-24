package utils

import (
	"context"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func ReadConfigFile(fileName string) *types.Project {
	projectName := "stack-stitcher"
	ctx := context.Background()
	workingDir, wdErr := os.Getwd()
	check(wdErr)

	path := filepath.Join(workingDir, fileName)
	options, projectErr := cli.NewProjectOptions(
		[]string{path},
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithName(projectName),
	)
	check(projectErr)

	project, loadErr := options.LoadProject(ctx)
	check(loadErr)

	return project
}

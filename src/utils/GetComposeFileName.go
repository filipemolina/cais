package utils

import (
	"errors"
	"fmt"
	"os"
)

// ErrNoComposeFile is returned by GetComposeFileName when none of the
// candidate file names exist in the current directory. The bootstrap flow
// uses it to distinguish "no file yet" from other load errors and offer to
// create one.
var ErrNoComposeFile = errors.New("no compose file found in the current directory")

// configFileNames are the names Docker tries, in the order it tries them:
// the first one that exists in the directory wins. The order is fixed and
// identical to Docker's on purpose - every docker invocation in this app
// shells out without -f and lets Docker resolve the file itself, so the UI
// has to agree with Docker about what won.
var configFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

// GetComposeFileName resolves which compose file Docker would use in the
// current directory. It returns the winner plus every candidate that exists,
// in priority order (so the winner is candidates[0]), because the winner is
// the whole story only when it is the only one: with several present, the UI
// can say which others were in the running.
func GetComposeFileName() (winner string, candidates []string, err error) {
	files, err := os.ReadDir(".")
	if err != nil {
		return "", nil, fmt.Errorf("failed reading the current directory: %w", err)
	}

	curDirFileNames := make(map[string]struct{})
	for _, file := range files {
		if !file.IsDir() {
			curDirFileNames[file.Name()] = struct{}{}
		}
	}

	for _, fileName := range configFileNames {
		if _, ok := curDirFileNames[fileName]; ok {
			candidates = append(candidates, fileName)
		}
	}

	if len(candidates) == 0 {
		return "", nil, ErrNoComposeFile
	}

	return candidates[0], candidates, nil
}

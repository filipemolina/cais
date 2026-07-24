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

func GetComposeFileName() (string, error) {
	files, err := os.ReadDir(".")

	configFileNames := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}

	if err != nil {
		return "", fmt.Errorf("failed reading the current directory: %w", err)
	}

	curDirFileNames := make(map[string]struct{})

	for _, file := range files {
		if !file.IsDir() {
			curDirFileNames[file.Name()] = struct{}{}
		}
	}

	for _, fileName := range configFileNames {
		if _, ok := curDirFileNames[fileName]; ok {
			return fileName, nil
		}
	}

	return "", ErrNoComposeFile
}

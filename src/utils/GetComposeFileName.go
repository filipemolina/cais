package utils

import (
	"fmt"
	"os"
)

func GetComposeFileName() (string, error) {
	files, err := os.ReadDir(".")
	var mainConfigFile string

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

	// Populates the curDirFileNames with all file names
	// in the current directory
	for _, file := range files {
		if !file.IsDir() {
			curDirFileNames[file.Name()] = struct{}{}
		}
	}

	// Checks for the existence of a configFileName in the
	// curDirFileNames map and returns the first one found.
	for _, fileName := range configFileNames {
		if _, ok := curDirFileNames[fileName]; ok {
			mainConfigFile = fileName
			break
		}
	}

	if mainConfigFile == "" {
		return "", fmt.Errorf("no compose.yaml, compose.yml, docker-compose.yaml or docker-compose.yml found in the current directory")
	}

	return mainConfigFile, nil
}

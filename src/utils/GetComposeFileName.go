package utils

import (
	"log"
	"os"
)

func GetComposeFileName() string {
	files, err := os.ReadDir(".")
	var mainConfigFile string

	configFileNames := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}

	if err != nil {
		log.Fatalf("There has been an error while reading the files in the directory: %v", err)
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

	return mainConfigFile
}

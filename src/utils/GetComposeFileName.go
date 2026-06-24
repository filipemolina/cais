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

	for _, file := range files {
		if !file.IsDir() {
			curDirFileNames[file.Name()] = struct{}{}
		}
	}

	for _, fileName := range configFileNames {
		if _, ok := curDirFileNames[fileName]; ok {
			mainConfigFile = fileName
			break
		}
	}

	return mainConfigFile
}

package utils

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// ListComposeFiles returns the names of the YAML files in dir, sorted. It is
// the directory scan behind the Files page's file picker: any *.yaml or
// *.yml file is a candidate, because --file accepts any name, not just the
// four auto-detected ones - the picker is a way to choose, not a resolution
// order, so it is not limited to GetComposeFileName's canonical names.
func ListComposeFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed reading %s: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			files = append(files, name)
		}
	}

	sort.Strings(files)
	return files, nil
}

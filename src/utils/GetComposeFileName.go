package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoComposeFile is returned by GetComposeFileName when none of the
// candidate file names exist in the directory it looked in. The bootstrap flow
// uses it to distinguish "no file yet" from other load errors and offer to
// create one.
var ErrNoComposeFile = errors.New("no compose file found")

// ComposeSource is where the app was told to look for a compose file: the
// zero value is "resolve one in the current directory", which is what it does
// when no flag is given.
//
// File and Dir never both apply - main.go rejects the combination - so this
// is two spellings of one answer rather than a search path.
type ComposeSource struct {
	// File is an exact path from --file. It skips resolution entirely: the
	// user named the file, so there is nothing to pick between.
	File string
	// Dir is the directory from --dir to resolve in. Empty means the current
	// directory.
	Dir string
}

// configFileNames are the names Docker tries, in the order it tries them:
// the first one that exists in the directory wins. The order is fixed and
// identical to Docker's on purpose - the app resolves the file and then
// passes it to docker as --file (see ComposeFileArgs), so agreeing with what
// docker would have picked on its own keeps `--dir` and a bare run identical.
var configFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

// GetComposeFileName resolves which compose file to use for source. It returns
// the winner plus every candidate that exists, in priority order (so the
// winner is candidates[0]), because the winner is the whole story only when it
// is the only one: with several present, the UI can say which others were in
// the running.
//
// Paths are returned as they will be used: joined with source.Dir, so the
// result can go straight to docker, to the YAML writers and to the footer.
func GetComposeFileName(source ComposeSource) (winner string, candidates []string, err error) {
	// An explicit --file is its own answer, and the only candidate. main.go
	// has already checked it exists, so failing to read it now is an ordinary
	// load error rather than the bootstrap state.
	if source.File != "" {
		return source.File, []string{source.File}, nil
	}

	dir := source.Dir
	if dir == "" {
		dir = "."
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, fmt.Errorf("failed reading %s: %w", dir, err)
	}

	dirFileNames := make(map[string]struct{})
	for _, file := range files {
		if !file.IsDir() {
			dirFileNames[file.Name()] = struct{}{}
		}
	}

	for _, fileName := range configFileNames {
		if _, ok := dirFileNames[fileName]; ok {
			// Join with source.Dir rather than dir: the fallback "." above is
			// for reading the directory, and prefixing every path with ./
			// would show up in the footer.
			candidates = append(candidates, filepath.Join(source.Dir, fileName))
		}
	}

	if len(candidates) == 0 {
		return "", nil, fmt.Errorf("%w in %s", ErrNoComposeFile, dir)
	}

	return candidates[0], candidates, nil
}

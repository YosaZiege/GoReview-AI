package python

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

var skipDirs = map[string]bool{
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".git":         true,
	"node_modules": true,
	".tox":         true,
	"dist":         true,
	"build":        true,
	".eggs":        true,
}

// WalkPyFiles returns all .py file paths under root, respecting .gitignore if present.
func WalkPyFiles(root string) ([]string, error) {
	var gi *ignore.GitIgnore
	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err == nil {
		gi, _ = ignore.CompileIgnoreFile(filepath.Join(root, ".gitignore"))
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] || strings.HasSuffix(d.Name(), ".egg-info") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".py") {
			return nil
		}
		if gi != nil {
			if rel, err := filepath.Rel(root, path); err == nil && gi.MatchesPath(rel) {
				return nil
			}
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

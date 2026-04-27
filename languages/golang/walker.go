package golang

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// skipDirs are always excluded regardless of .gitignore.
var skipDirs = map[string]bool{
	"vendor":       true,
	"testdata":     true,
	".git":         true,
	"node_modules": true,
}

// WalkGoFiles returns all .go file paths under root, respecting .gitignore if present.
func WalkGoFiles(root string) ([]string, error) {
	var gitignore *ignore.GitIgnore
	gitignorePath := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		gitignore, _ = ignore.CompileIgnoreFile(gitignorePath)
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Check .gitignore — use path relative to root for matching.
		if gitignore != nil {
			rel, err := filepath.Rel(root, path)
			if err == nil && gitignore.MatchesPath(rel) {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})
	return files, err
}

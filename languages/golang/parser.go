package golang

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"

	"golang.org/x/tools/go/packages"
)

// ParsedFile holds the full parsed state for a single .go source file.
type ParsedFile struct {
	Path   string
	Fset   *token.FileSet
	File   *ast.File
	Source []byte         // raw source bytes
	Info   *types.Info    // nil if type-checking failed
	Pkg    *types.Package
}

// ParseDir loads every Go package reachable from dir using the Go toolchain.
// It provides full type information for packages that compile cleanly and
// degrades gracefully (Info == nil) for packages with unresolvable imports.
func ParseDir(dir string) ([]ParsedFile, error) {
	fset := token.NewFileSet()

	cfg := &packages.Config{
		Dir:  dir,
		Fset: fset,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		// Type errors are reported per-package in pkg.Errors — we handle them below.
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages from %s: %w", dir, err)
	}

	var results []ParsedFile

	for _, pkg := range pkgs {
		// If any type errors exist, treat type info as unreliable — set to nil
		// so detectors fall back to AST-only heuristics rather than wrong answers.
		typeInfo := pkg.TypesInfo
		typesPkg := pkg.Types
		for _, e := range pkg.Errors {
			if e.Kind == packages.TypeError {
				typeInfo = nil
				typesPkg = nil
				break
			}
		}

		for _, astFile := range pkg.Syntax {
			path := fset.File(astFile.Pos()).Name()

			src, err := os.ReadFile(path)
			if err != nil {
				// File is in memory but unreadable on disk — skip gracefully.
				continue
			}

			results = append(results, ParsedFile{
				Path:   path,
				Fset:   fset,
				File:   astFile,
				Source: src,
				Info:   typeInfo,
				Pkg:    typesPkg,
			})
		}
	}

	return results, nil
}

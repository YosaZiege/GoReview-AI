package core

// PackageSketch is a stripped-source representation of all files in one package
// directory, used for package-level LLM deep analysis.
// Function bodies are removed; signatures, types, and imports are preserved.
type PackageSketch struct {
	Dir      string   // package directory path
	Language string   // "go" or "python"
	Files    []string // file paths included in this sketch
	Source   string   // concatenated stripped source with per-file headers
}

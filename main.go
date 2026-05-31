package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/detectors"
	"github.com/yourorg/gorview/detectors/complexity"
	"github.com/yourorg/gorview/detectors/concretedep"
	"github.com/yourorg/gorview/detectors/godstruct"
	pydetectors "github.com/yourorg/gorview/detectors/python"
	pycomplexity "github.com/yourorg/gorview/detectors/python/complexity"
	pygodclass "github.com/yourorg/gorview/detectors/python/godclass"
	"github.com/yourorg/gorview/languages/golang"
	"github.com/yourorg/gorview/languages/python"
	"github.com/yourorg/gorview/llm"
	ollamaLLM "github.com/yourorg/gorview/llm/ollama"
	openaiLLM "github.com/yourorg/gorview/llm/openai"
	"github.com/yourorg/gorview/output"
)
// Final modifs
var (
	flagFormat       string
	flagOutputFile   string
	flagEnableLLM    bool
	flagDeepAnalysis bool
	flagDeepTimeout  int
	flagLLMProvider  string
	flagLLMModel     string
	flagLLMBaseURL   string
	flagLLMAPIKey    string
	flagMinScore     int
	flagTimeout      int
)

func main() {
	root := &cobra.Command{
		Use:   "gorview [dir]",
		Short: "GoReview AI — multi-language architectural smell detector",
		Long: `GoReview AI analyses source code for architectural smells and suggests
design patterns to address them. Supports Go and Python. Run without --llm for
instant, fully local analysis.`,
		Args: cobra.MaximumNArgs(1),
		RunE: run,
	}

	root.Flags().StringVarP(&flagFormat, "format", "f", "terminal", "output format: terminal, json, html")
	root.Flags().StringVarP(&flagOutputFile, "output", "o", "", "write report to file instead of stdout")
	root.Flags().BoolVar(&flagEnableLLM, "llm", false, "enable LLM enrichment (adds explanations and refactoring examples)")
	root.Flags().BoolVar(&flagDeepAnalysis, "deep", false, "deep LLM analysis — sends stripped package skeletons, one call per package")
	root.Flags().IntVar(&flagDeepTimeout, "deep-timeout", 300, "per-package LLM timeout for deep analysis in seconds (local models need more time)")
	root.Flags().StringVar(&flagLLMProvider, "llm-provider", "ollama", "LLM provider: ollama, openai")
	root.Flags().StringVar(&flagLLMModel, "llm-model", "qwen2.5-coder:7b", "LLM model name")
	root.Flags().StringVar(&flagLLMBaseURL, "llm-base-url", "", "Ollama base URL (default: http://localhost:11434)")
	root.Flags().StringVar(&flagLLMAPIKey, "llm-api-key", "", "OpenAI API key (or set OPENAI_API_KEY)")
	root.Flags().IntVar(&flagMinScore, "min-score", 0, "exit 1 if maintainability score is below this threshold (CI mode)")
	root.Flags().IntVar(&flagTimeout, "timeout", 600, "static analysis + enrichment timeout in seconds")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(_ *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	var allFindings []core.Finding

	// --- Go analysis ---
	goFiles, err := golang.ParseDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gorview: Go parse error: %v\n", err)
	} else if len(goFiles) > 0 {
		goDetectors := []detectors.Detector{
			godstruct.Detector{},
			concretedep.Detector{},
			complexity.Detector{},
		}
		allFindings = append(allFindings, detectors.RunAll(ctx, goDetectors, goFiles)...)
	}

	// --- Python analysis ---
	pyFiles, err := python.ParseDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gorview: Python parse error: %v\n", err)
	} else if len(pyFiles) > 0 {
		pyDs := []pydetectors.Detector{
			pygodclass.Detector{},
			pycomplexity.Detector{},
		}
		allFindings = append(allFindings, pydetectors.RunAll(ctx, pyDs, pyFiles)...)
	}

	if len(goFiles) == 0 && len(pyFiles) == 0 {
		fmt.Fprintf(os.Stderr, "gorview: no supported source files found in %s\n", dir)
		return nil
	}

	// --deep implies --llm
	if flagDeepAnalysis && !flagEnableLLM {
		fmt.Fprintln(os.Stderr, "gorview: --deep requires --llm; enabling LLM automatically")
		flagEnableLLM = true
	}

	// LLM enrichment + deep analysis (optional)
	if flagEnableLLM {
		provider, err := buildProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gorview: LLM setup failed: %v\n", err)
		} else {
			allFindings, err = provider.Enrich(ctx, allFindings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gorview: LLM enrichment failed: %v\n", err)
			}
			if flagDeepAnalysis {
				if analyser, ok := provider.(llm.DeepAnalyser); ok {
					sketches := buildGoSketches(goFiles)
					sketches = append(sketches, buildPySketches(pyFiles)...)
					allFindings = runDeepAnalysis(ctx, analyser, sketches, allFindings)
				} else {
					fmt.Fprintf(os.Stderr, "gorview: deep analysis not supported by provider %q\n", flagLLMProvider)
				}
			}
		}
	}

	report := core.NewReport(dir, allFindings)

	// Output
	var w io.Writer = os.Stdout
	if flagOutputFile != "" {
		f, err := os.Create(flagOutputFile)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	switch flagFormat {
	case "json":
		if err := output.PrintJSON(w, report); err != nil {
			return err
		}
	case "html":
		if err := output.PrintHTML(w, report); err != nil {
			return err
		}
	default:
		output.PrintTerminal(w, report)
	}

	// CI mode: fail if score is below threshold
	if flagMinScore > 0 && report.Score < flagMinScore {
		fmt.Fprintf(os.Stderr, "gorview: score %d < min-score %d\n", report.Score, flagMinScore)
		os.Exit(1)
	}

	return nil
}

// buildGoSketches groups non-generated Go files by package directory and
// produces one stripped PackageSketch per directory.
func buildGoSketches(files []golang.ParsedFile) []core.PackageSketch {
	groups := map[string][]golang.ParsedFile{}
	for _, pf := range files {
		// Skip auto-generated files (they add noise, not architectural signal).
		header := pf.Source
		if len(header) > 512 {
			header = header[:512]
		}
		if bytes.Contains(header, []byte("Code generated")) {
			continue
		}
		if strings.Contains(filepath.ToSlash(pf.Path), "/generated/") ||
			strings.Contains(filepath.ToSlash(pf.Path), "/gen/") {
			continue
		}
		dir := filepath.Dir(pf.Path)
		groups[dir] = append(groups[dir], pf)
	}

	var sketches []core.PackageSketch
	for dir, grp := range groups {
		var sb strings.Builder
		var paths []string
		for _, pf := range grp {
			fmt.Fprintf(&sb, "// === %s ===\n\n", pf.Path)
			sb.WriteString(golang.StripSource(pf))
			sb.WriteString("\n\n")
			paths = append(paths, pf.Path)
		}
		sketches = append(sketches, core.PackageSketch{
			Dir:      dir,
			Language: "go",
			Files:    paths,
			Source:   sb.String(),
		})
	}
	return sketches
}

// buildPySketches groups Python files by directory and produces one stripped
// PackageSketch per directory.
func buildPySketches(files []python.ParsedFile) []core.PackageSketch {
	groups := map[string][]python.ParsedFile{}
	for _, pf := range files {
		dir := filepath.Dir(pf.Path)
		groups[dir] = append(groups[dir], pf)
	}

	var sketches []core.PackageSketch
	for dir, grp := range groups {
		var sb strings.Builder
		var paths []string
		for _, pf := range grp {
			fmt.Fprintf(&sb, "# === %s ===\n\n", pf.Path)
			sb.WriteString(python.StripSource(pf))
			sb.WriteString("\n\n")
			paths = append(paths, pf.Path)
		}
		sketches = append(sketches, core.PackageSketch{
			Dir:      dir,
			Language: "python",
			Files:    paths,
			Source:   sb.String(),
		})
	}
	return sketches
}

// runDeepAnalysis sends each package sketch to the LLM analyser.
// Each sketch gets its own timeout so one slow package cannot block others.
// The parent context is checked between packages to respect Ctrl-C.
func runDeepAnalysis(parent context.Context, analyser llm.DeepAnalyser, sketches []core.PackageSketch, existing []core.Finding) []core.Finding {
	all := existing
	timeout := time.Duration(flagDeepTimeout) * time.Second
	for _, sketch := range sketches {
		select {
		case <-parent.Done():
			fmt.Fprintln(os.Stderr, "gorview: deep analysis cancelled")
			return all
		default:
		}
		pkgCtx, cancel := context.WithTimeout(parent, timeout)
		fmt.Fprintf(os.Stderr, "[gorview] deep: analysing %s ...\n", sketch.Dir)
		found, err := analyser.AnalysePackage(pkgCtx, sketch, all)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gorview: deep analysis %s: %v\n", sketch.Dir, err)
			continue
		}
		all = append(all, found...)
	}
	return all
}

func buildProvider() (llm.Provider, error) {
	apiKey := flagLLMAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	switch flagLLMProvider {
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key required (--llm-api-key or OPENAI_API_KEY)")
		}
		return openaiLLM.New(apiKey, flagLLMModel), nil
	case "ollama":
		return ollamaLLM.New(flagLLMBaseURL, flagLLMModel), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider %q (use: ollama, openai)", flagLLMProvider)
	}
}

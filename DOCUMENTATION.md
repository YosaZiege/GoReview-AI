# GoReview AI — Documentation

GoReview AI (`gorview`) is a command-line tool that analyses Go and Python source code for architectural smells — patterns in the code structure that make it harder to maintain, test, or extend. It runs instantly with no dependencies (static mode), and optionally connects to a local or cloud LLM to add explanations, refactoring examples, and deeper cross-file insights.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [How It Works — Overview](#how-it-works--overview)
3. [Analysis Modes](#analysis-modes)
4. [Flags Reference](#flags-reference)
5. [Output Formats](#output-formats)
6. [Scoring System](#scoring-system)
7. [Architectural Smells Detected](#architectural-smells-detected)
8. [LLM Integration](#llm-integration)
9. [CI / Automation](#ci--automation)
10. [Project Structure](#project-structure)

---

## Quick Start

```bash
# Static analysis of the current directory
./main .

# Static analysis of a specific project
./main /path/to/your/project

# With LLM explanations and refactoring examples (uses local Ollama)
./main --llm .

# Full deep analysis — LLM reads entire package skeletons
./main --deep .

# Save report to an HTML file
./main -f html -o report.html .

# CI mode — fail with exit code 1 if score drops below 70
./main --min-score 70 .
```

---

## How It Works — Overview

Gorview runs in two stages:

**Stage 1 — Static Analysis (always runs)**
The tool parses every `.go` and `.py` file in the target directory (recursively). It builds an in-memory representation of the code structure — types, methods, functions, fields, imports — and runs a set of detectors against it. Each detector looks for one specific architectural smell. This stage is instant, produces no network traffic, and requires no API keys.

**Stage 2 — LLM Enrichment (optional, `--llm` or `--deep`)**
If an LLM is enabled, it operates on top of the static results:

- **`--llm` mode**: For each finding from Stage 1, the LLM receives a compact summary (smell type, component name, location, metrics) and returns a plain-English explanation plus a Before/After refactoring example.
- **`--deep` mode**: In addition to enriching existing findings, the LLM receives the full skeleton of each package — all files stripped of their function bodies, leaving only signatures, types, and imports. It then performs its own cross-file architectural review and can surface smells the static detectors cannot see.

The two stages are additive: `--deep` always includes the `--llm` enrichment pass first.

---

## Analysis Modes

| Command | What happens |
|---|---|
| `./main .` | Static detectors only. Instant. No network. |
| `./main --llm .` | Static detectors + LLM explanation for each finding. |
| `./main --deep .` | Static detectors + LLM explanation + LLM package-level deep scan. |

### Static mode
No configuration needed. Gorview walks the directory tree, parses all Go and Python files, and prints a sorted, coloured report.

### LLM enrichment mode (`--llm`)
Each static finding is sent to the LLM one at a time. The LLM receives only the structural description — never raw source code — and returns a diagnosis and refactoring sketch. If the LLM fails for a particular finding, that finding is still shown but without the enrichment fields.

### Deep analysis mode (`--deep`)
`--deep` automatically enables `--llm` if you forget to pass it. After the enrichment pass, gorview groups all source files by their package directory, strips function bodies from each file (keeping only signatures, type definitions, and imports), and sends the combined skeleton of each package to the LLM as a single prompt. The LLM can then detect cross-file patterns — for example, a function in one file that is clearly more interested in the internals of a type defined in another file (Feature Envy), or a layer boundary being crossed through a chain of calls spread across multiple files.

Auto-generated files (those with `Code generated` in their header, or located under `/generated/` or `/gen/` directories) are excluded from the deep analysis sketches to reduce noise.

---

## Flags Reference

### Target

| Flag | Default | Description |
|---|---|---|
| `[dir]` | `.` | Directory to analyse. Positional argument, not a flag. Gorview walks it recursively. |

### Output

| Flag | Short | Default | Description |
|---|---|---|---|
| `--format` | `-f` | `terminal` | Output format. Options: `terminal`, `json`, `html`. |
| `--output` | `-o` | _(stdout)_ | Write the report to a file instead of printing to the terminal. |

### LLM

| Flag | Default | Description |
|---|---|---|
| `--llm` | `false` | Enable LLM enrichment. Adds a plain-English explanation and Before/After refactoring example to each static finding. |
| `--deep` | `false` | Enable deep LLM analysis. Sends stripped package skeletons to the LLM for cross-file architectural review. Automatically enables `--llm`. |
| `--llm-provider` | `ollama` | LLM backend to use. Options: `ollama` (local, free), `openai` (cloud, requires API key). |
| `--llm-model` | `qwen2.5-coder:7b` | Model name to use. For Ollama: any model name you have pulled (e.g. `llama3.2:3b`, `codellama:7b`). For OpenAI: e.g. `gpt-4o`, `gpt-4o-mini`. |
| `--llm-base-url` | `http://localhost:11434` | Ollama server URL. Change this if Ollama runs on a different host or port. |
| `--llm-api-key` | _(env)_ | OpenAI API key. Can also be set via the `OPENAI_API_KEY` environment variable. |

### Timeouts

| Flag | Default | Description |
|---|---|---|
| `--timeout` | `600` | Total timeout in seconds for the entire run (static analysis + all LLM calls). |
| `--deep-timeout` | `300` | Per-package timeout in seconds for deep analysis. Local models on CPU can be slow; increase this if you see timeout errors. |

### CI

| Flag | Default | Description |
|---|---|---|
| `--min-score` | `0` (disabled) | If the maintainability score falls below this value, gorview exits with code `1`. Use in CI pipelines to enforce a minimum quality bar. |

---

## Output Formats

### terminal (default)
Coloured, human-readable output printed to stdout. Findings are sorted by severity (critical first). Each finding shows:
- Severity level and smell type
- File path and line number
- Component name (struct, function, or method)
- Suggested design pattern and estimated refactoring effort
- LLM explanation (if `--llm` was used)
- Before/After refactoring sketch (if `--llm` was used)

At the bottom: a maintainability score out of 100 and a count per severity level.

### json
Machine-readable JSON. Useful for piping results into other tools or dashboards. The top-level object contains the target directory, the score, and an array of findings.

### html
A self-contained HTML file with the same information as the terminal output, suitable for sharing or archiving reports.

---

## Scoring System

Gorview calculates a maintainability score from 0 to 100, starting at 100 and deducting points for each finding:

| Severity | Label | Points deducted |
|---|---|---|
| Critical | `CRITIQUE` | −15 |
| Medium | `MOYEN` | −7 |
| Low | `FAIBLE` | −3 |

The score cannot go below 0. A score of 80 or above is generally considered healthy. The `--min-score` flag lets you use this in CI to block merges on low-quality code.

---

## Architectural Smells Detected

### Go detectors

**`god_struct`** — God Object
A struct with too many fields or too many methods is taking on too many responsibilities. It becomes hard to test because it touches too many things, and hard to change without breaking unrelated behaviour.
- Flagged when: `fields > 10` or `methods > 15`
- Critical when: `fields > 20` or `methods > 30`
- Suggested pattern: Façade (break it into focused sub-components)

**`concrete_dep`** — Concrete Dependency
A struct field that holds a concrete type (e.g. a database driver, an HTTP client, a specific service implementation) where an interface should be used. This couples the struct tightly to one implementation, making it impossible to swap out for testing or for future changes.
- Detected on fields with conventional dependency names (`db`, `store`, `repo`, `svc`, `client`, `cache`, `queue`, `logger`, etc.)
- Suggested pattern: Dependency Injection via interface

**`high_complexity`** — High Cyclomatic Complexity
A function or method with too many branching paths (if/else, switch cases, loops, error checks). High complexity means the function is hard to read, hard to test (you need a test case for every branch), and likely to harbour bugs.
- Flagged when: cyclomatic complexity > 10
- Critical when: CC > 20
- Suggested pattern: Strategy (extract branches into separate strategy objects or functions)

### Python detectors

**`god_class`** — God Class
Same concept as `god_struct` but for Python classes. A class with too many methods or attributes is doing too much.
- Flagged when: `methods > 15` or `fields > 10`
- Suggested pattern: Façade

**`high_complexity`** — High Cyclomatic Complexity
Same as the Go version but applied to Python functions and methods.
- Flagged when: cyclomatic complexity > 10
- Suggested pattern: Strategy

### LLM-only smells (deep mode)

These are detected only in `--deep` mode, because they require reading multiple files or understanding intent:

- **Feature Envy** — A function is more interested in the data of another module than its own.
- **Layer Violation** — Code in one architectural layer (e.g. a business logic struct) directly calls into a layer it shouldn't know about (e.g. database drivers, HTTP handlers).
- **Missing Abstraction** — A repeated pattern or a complex operation that would benefit from being named and encapsulated behind an interface.
- **Temporal Coupling** — Two functions that must always be called in a specific order, but nothing in the code enforces that order.
- **Anemic Domain Model** — Domain objects that are purely data containers with no behaviour, while all the logic lives in separate service layers.

---

## LLM Integration

### Using Ollama (local, free)

Ollama runs models on your own machine. No API key, no data leaving your network.

```bash
# Ollama is started automatically if it is not running.
# The model is pulled automatically on first use.
./main --llm --llm-provider ollama --llm-model qwen2.5-coder:7b .
```

The default model is `qwen2.5-coder:7b`, which is optimised for code analysis and runs well on a machine with 8 GB of RAM or more. If you want a faster but less capable model, use `llama3.2:3b`. If Ollama is not installed, install it with:

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

Then pull a model:

```bash
ollama pull qwen2.5-coder:7b
```

### Using OpenAI

```bash
export OPENAI_API_KEY=sk-...
./main --llm --llm-provider openai --llm-model gpt-4o-mini .
```

Or pass the key directly:

```bash
./main --llm --llm-provider openai --llm-model gpt-4o-mini --llm-api-key sk-... .
```

### What the LLM sees

The LLM is never shown raw source code. In enrichment mode it receives a compact structural description of each finding (smell type, component name, location, metrics). In deep mode it receives a skeleton of the package where function bodies have been removed — only signatures, type definitions, and imports remain. This keeps token usage low while giving the LLM enough context to reason about architecture.

---

## CI / Automation

```yaml
# GitHub Actions example
- name: Architectural review
  run: ./main --min-score 70 --format json -o gorview-report.json .

- name: Upload report
  uses: actions/upload-artifact@v4
  with:
    name: gorview-report
    path: gorview-report.json
```

```bash
# Pre-commit hook example
./main --min-score 60 . || { echo "gorview: score too low"; exit 1; }
```

When `--min-score` is set and the score falls below the threshold, gorview prints the actual vs threshold score to stderr and exits with code `1`. All output is still produced normally before the exit.

---

## Project Structure

```
GoReview-AI/
├── main.go                         Entry point and CLI
├── core/                           Shared data types
├── detectors/                      Static smell detectors
│   ├── complexity/                 Go cyclomatic complexity detector
│   ├── concretedep/                Go concrete dependency detector
│   ├── godstruct/                  Go god struct detector
│   └── python/                     Python detectors
│       ├── complexity/             Python cyclomatic complexity detector
│       └── godclass/               Python god class detector
├── languages/                      Language parsers and utilities
│   ├── golang/                     Go parser, source stripper, AST summariser
│   └── python/                     Python parser, source stripper, AST summariser
├── llm/                            LLM interfaces, prompt builders, response parser
│   ├── ollama/                     Ollama client (local LLM)
│   └── openai/                     OpenAI client (cloud LLM)
└── output/                         Report renderers (terminal, JSON, HTML)
```

### `main.go`
The entry point. Defines all CLI flags, orchestrates the full analysis pipeline: parse files → run static detectors → optionally enrich with LLM → optionally run deep analysis → render and write the report. Also handles the `--min-score` CI exit code.

### `core/`
The shared data types used by every other package. Contains the `Finding` struct (one detected smell), the `Report` struct (collection of findings for a directory with a score), the scoring formula, the `PackageSketch` type (stripped source sent to the LLM for deep analysis), and the `ASTSummary` type (structured AST representation used in the legacy per-file analysis path).

### `detectors/`
Each sub-package is one detector. A detector receives a list of parsed files and returns a list of `Finding` values — it does not know about the LLM or the output format. Go detectors and Python detectors use separate interfaces because they receive different parsed file types. The `detector.go` file at the top level provides the `RunAll` helper that calls every registered detector and collects the results.

### `detectors/godstruct/`
Flags Go structs that have more than 10 fields or more than 15 methods. Counts fields and methods separately across all files in a package so it catches methods declared in different files for the same type.

### `detectors/complexity/`
Computes the cyclomatic complexity of every Go function and method by counting branching nodes in the AST (if, for, switch cases, select, &&, ||). Flags anything above 10. Above 20 is critical.

### `detectors/concretedep/`
Walks Go struct fields and checks whether fields with conventional dependency names (db, repo, service, client, etc.) hold concrete types instead of interfaces. A concrete type is one that is not an interface, is not a primitive, and is not from the standard library.

### `detectors/python/godclass/`
Same logic as the Go `godstruct` detector but operates on Python class information extracted by the Python parser. Counts methods and instance attributes (fields set via `self.x = ...` in `__init__`).

### `detectors/python/complexity/`
Same as the Go complexity detector but applied to Python. Cyclomatic complexity is estimated by counting branching keywords (`if`, `elif`, `for`, `while`, `except`, `and`, `or`) in the function source text.

### `languages/golang/`
Everything needed to load and understand Go source files. The parser walks a directory recursively and returns typed, AST-parsed file objects. The source stripper removes function bodies from files so only skeletons remain — used for the deep LLM prompt. The summariser converts the AST into a compact JSON-friendly `ASTSummary` structure. The walker is a utility for traversing AST nodes.

### `languages/python/`
Same role as the Go languages package but for Python. Because Python has no standard compile-time AST library in Go, the parser uses a combination of regex and line-by-line text analysis to extract class definitions, method signatures, function definitions, and complexity estimates. The stripper removes function bodies for deep analysis.

### `llm/`
The LLM abstraction layer. `provider.go` defines the `Provider` interface (one method: `Enrich`). `deep_analyser.go` defines the `DeepAnalyser` interface (two methods: `Analyse` for the legacy per-file path, `AnalysePackage` for the preferred package-level path) and the `ParseDeepFindings` function that extracts a JSON array of findings from any LLM response — handling cases where the model wraps its output in prose or markdown. `context_builder.go` builds the enrichment prompt for a single finding. `deep_prompt.go` builds both the legacy AST prompt and the package skeleton prompt for deep analysis.

### `llm/ollama/`
The Ollama client. Implements both `Provider` (Enrich) and `DeepAnalyser` (Analyse, AnalysePackage). On every call it checks whether Ollama is running and whether the requested model is present, starting the server and pulling the model automatically if needed. Uses a 300-second HTTP timeout to accommodate slow CPU inference.

### `llm/openai/`
The OpenAI client. Same interface as the Ollama client. Sends requests to the OpenAI Chat Completions API. Does not auto-install anything — requires a valid API key.

### `output/`
Three renderers, all receiving the same `Report` type. The terminal renderer sorts findings by severity and uses colour. The JSON renderer produces a structured JSON object. The HTML renderer produces a self-contained HTML file. All three respect the `--output` flag for writing to a file.

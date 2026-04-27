package python

import (
	"os"
	"strings"
)

// StripSource returns the Python file's source with all function bodies replaced
// by a single `...` placeholder, preserving class/def signatures, decorators,
// and imports so the LLM sees the architectural skeleton.
func StripSource(pf ParsedFile) string {
	data, err := os.ReadFile(pf.Path)
	if err != nil {
		return ""
	}
	return stripPython(string(data))
}

func stripPython(src string) string {
	lines := strings.Split(src, "\n")
	var out []string

	type funcFrame struct {
		defIndent       int
		placeholderDone bool
	}
	var funcStack []funcFrame

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Preserve blank lines only outside function bodies.
		if trimmed == "" {
			if len(funcStack) == 0 {
				out = append(out, line)
			}
			continue
		}

		indent := indentOf(line) // reuses the unexported helper from parser.go

		// Pop any function frames we've outdented past.
		for len(funcStack) > 0 && indent <= funcStack[len(funcStack)-1].defIndent {
			funcStack = funcStack[:len(funcStack)-1]
		}

		inBody := len(funcStack) > 0
		isDef := strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ")
		isClass := strings.HasPrefix(trimmed, "class ")
		isDecorator := strings.HasPrefix(trimmed, "@")

		if inBody {
			top := &funcStack[len(funcStack)-1]
			// Emit the placeholder once for this function's body.
			if !top.placeholderDone {
				placeholder := strings.Repeat(" ", top.defIndent+4) + "..."
				out = append(out, placeholder)
				top.placeholderDone = true
			}
			// Nested def/class lines are still architectural signal — keep them.
			if isDef || isClass || isDecorator {
				out = append(out, line)
				if isDef {
					funcStack = append(funcStack, funcFrame{defIndent: indent})
				}
			}
			// All other body lines are silently dropped.
		} else {
			out = append(out, line)
			if isDef {
				funcStack = append(funcStack, funcFrame{defIndent: indent})
			}
		}
	}

	return strings.Join(out, "\n")
}

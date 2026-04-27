package golang

import (
	"fmt"
	"go/ast"
	"sort"
)

// StripSource returns the file's source with all function bodies replaced by
// `{ /* N lines */ }`, preserving package declarations, imports, type
// definitions, and function signatures — everything the LLM needs for
// architectural analysis at a fraction of the token cost.
func StripSource(pf ParsedFile) string {
	type span struct {
		start, end int // byte offsets (Lbrace … Rbrace inclusive)
		lines      int // number of body lines stripped
	}

	var spans []span
	for _, decl := range pf.File.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		lb := pf.Fset.Position(fd.Body.Lbrace)
		rb := pf.Fset.Position(fd.Body.Rbrace)
		bodyLines := rb.Line - lb.Line - 1
		if bodyLines <= 0 {
			continue // empty body — leave as-is
		}
		spans = append(spans, span{
			start: lb.Offset,
			end:   rb.Offset,
			lines: bodyLines,
		})
	}

	// Process in reverse order so byte offsets stay valid after each replacement.
	sort.Slice(spans, func(i, j int) bool { return spans[i].start > spans[j].start })

	src := make([]byte, len(pf.Source))
	copy(src, pf.Source)

	for _, s := range spans {
		repl := []byte(fmt.Sprintf("{ /* %d lines */ }", s.lines))
		var next []byte
		next = append(next, src[:s.start]...)
		next = append(next, repl...)
		next = append(next, src[s.end+1:]...)
		src = next
	}

	return string(src)
}

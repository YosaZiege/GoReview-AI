package python

import (
	"context"

	"github.com/yourorg/gorview/core"
	pythonlang "github.com/yourorg/gorview/languages/python"
)

// Detector analyses parsed Python files and returns architectural findings.
type Detector interface {
	Name() string
	Detect(files []pythonlang.ParsedFile) []core.Finding
}

// RunAll executes all Python detectors concurrently and merges findings.
func RunAll(ctx context.Context, ds []Detector, files []pythonlang.ParsedFile) []core.Finding {
	ch := make(chan []core.Finding, len(ds))
	for _, d := range ds {
		d := d
		go func() {
			select {
			case <-ctx.Done():
				ch <- nil
			default:
				ch <- d.Detect(files)
			}
		}()
	}
	var all []core.Finding
	for range ds {
		all = append(all, <-ch...)
	}
	return all
}

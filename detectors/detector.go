package detectors

import (
	"context"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/languages/golang"
)

// Detector analyses parsed Go files and returns architectural findings.
type Detector interface {
	Name() string
	Detect(files []golang.ParsedFile) []core.Finding
}

// RunAll executes all detectors concurrently and merges findings.
// Respects ctx cancellation — cancelled detectors contribute no findings.
func RunAll(ctx context.Context, ds []Detector, files []golang.ParsedFile) []core.Finding {
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

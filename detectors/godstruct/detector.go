package godstruct

import (
	"go/ast"
	"path/filepath"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/languages/golang"
)

const (
	fieldThreshold  = 10
	methodThreshold = 15
)

// Detector flags structs with too many fields or methods (God Object smell).
type Detector struct{}

func (Detector) Name() string { return "god_struct" }

func (Detector) Detect(files []golang.ParsedFile) []core.Finding {
	type meta struct {
		name    string
		fields  int
		methods int
		file    string
		line    int
	}

	// pkgID -> structName -> meta
	byPkg := map[string]map[string]*meta{}
	pkgOf := map[string]string{}

	pkgIDFor := func(pf golang.ParsedFile) string {
		if pf.Pkg != nil {
			return pf.Pkg.Path()
		}
		return filepath.Dir(pf.Path)
	}

	// Pass 1: collect struct field counts
	for _, pf := range files {
		id := pkgIDFor(pf)
		pkgOf[pf.Path] = id
		if byPkg[id] == nil {
			byPkg[id] = map[string]*meta{}
		}
		ast.Inspect(pf.File, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			pos := pf.Fset.Position(ts.Pos())
			m := &meta{name: ts.Name.Name, file: pf.Path, line: pos.Line}
			for _, f := range st.Fields.List {
				if len(f.Names) == 0 {
					m.fields++ // embedded field
				} else {
					m.fields += len(f.Names)
				}
			}
			byPkg[id][ts.Name.Name] = m
			return true
		})
	}

	// Pass 2: count methods per receiver type
	for _, pf := range files {
		id := pkgOf[pf.Path]
		for _, decl := range pf.File.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
				continue
			}
			name := recvTypeName(fd.Recv.List[0].Type)
			if m, ok := byPkg[id][name]; ok {
				m.methods++
			}
		}
	}

	var findings []core.Finding
	for _, pkg := range byPkg {
		for _, m := range pkg {
			if m.fields <= fieldThreshold && m.methods <= methodThreshold {
				continue
			}
			sev := core.SeverityMedium
			if m.fields > fieldThreshold*2 || m.methods > methodThreshold*2 {
				sev = core.SeverityCritical
			}
			findings = append(findings, core.Finding{
				Severity:  sev,
				SmellType: "god_struct",
				File:      m.file,
				Line:      m.line,
				Component: m.name,
				Metrics:   map[string]int{"fields": m.fields, "methods": m.methods},
				Pattern:   "Façade",
				Effort:    effortFor(m.fields, m.methods),
			})
		}
	}
	return findings
}

func recvTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvTypeName(t.X)
	}
	return ""
}

func effortFor(fields, methods int) core.EffortLevel {
	if fields > fieldThreshold*2 || methods > methodThreshold*2 {
		return core.EffortHigh
	}
	return core.EffortMedium
}

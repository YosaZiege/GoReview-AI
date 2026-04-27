package complexity

import (
	"go/ast"
	"go/token"

	"github.com/yourorg/gorview/core"
	"github.com/yourorg/gorview/languages/golang"
)

const ccThreshold = 10

// Detector flags functions/methods with cyclomatic complexity above the threshold.
type Detector struct{}

func (Detector) Name() string { return "high_complexity" }

func (Detector) Detect(files []golang.ParsedFile) []core.Finding {
	var findings []core.Finding
	for _, pf := range files {
		for _, decl := range pf.File.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			cc := cyclomaticComplexity(fd)
			if cc <= ccThreshold {
				continue
			}
			pos := pf.Fset.Position(fd.Pos())
			findings = append(findings, core.Finding{
				Severity:  severityForCC(cc),
				SmellType: "high_complexity",
				File:      pf.Path,
				Line:      pos.Line,
				Component: funcName(fd),
				Metrics:   map[string]int{"cc": cc},
				Pattern:   "Stratégie",
				Effort:    effortForCC(cc),
			})
		}
	}
	return findings
}

func cyclomaticComplexity(fd *ast.FuncDecl) int {
	cc := 1
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.IfStmt:
			cc++
		case *ast.ForStmt:
			cc++
		case *ast.RangeStmt:
			cc++
		case *ast.CaseClause:
			if v.List != nil {
				cc++
			}
		case *ast.CommClause:
			if v.Comm != nil {
				cc++
			}
		case *ast.BinaryExpr:
			if v.Op == token.LAND || v.Op == token.LOR {
				cc++
			}
		}
		return true
	})
	return cc
}

func funcName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	recv := recvTypeName(fd.Recv.List[0].Type)
	if recv == "" {
		return fd.Name.Name
	}
	return recv + "." + fd.Name.Name
}

func recvTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func severityForCC(cc int) core.Severity {
	switch {
	case cc > 20:
		return core.SeverityCritical
	case cc > 15:
		return core.SeverityMedium
	default:
		return core.SeverityLow
	}
}

func effortForCC(cc int) core.EffortLevel {
	switch {
	case cc > 20:
		return core.EffortHigh
	case cc > 15:
		return core.EffortMedium
	default:
		return core.EffortLow
	}
}

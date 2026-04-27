package python

import "github.com/yourorg/gorview/core"

// Summarize converts a ParsedFile into an ASTSummary for LLM deep analysis.
func Summarize(pf ParsedFile) core.ASTSummary {
	sum := core.ASTSummary{File: pf.Path}

	classIndex := map[string]int{}
	for _, ci := range pf.Classes {
		tn := core.TypeNode{
			Name: ci.Name,
			Kind: "class",
			Line: ci.Line,
		}
		classIndex[ci.Name] = len(sum.Types)
		sum.Types = append(sum.Types, tn)
	}

	for _, fi := range pf.Funcs {
		if fi.Class != "" {
			if idx, ok := classIndex[fi.Class]; ok {
				sum.Types[idx].Methods = append(sum.Types[idx].Methods, core.MethodNode{
					Name:       fi.Name,
					Line:       fi.Line,
					Complexity: fi.Complexity,
				})
			}
		} else {
			sum.Funcs = append(sum.Funcs, core.FuncNode{
				Name:       fi.Name,
				Line:       fi.Line,
				Complexity: fi.Complexity,
			})
		}
	}

	return sum
}

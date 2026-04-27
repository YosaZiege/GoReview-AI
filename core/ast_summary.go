package core

// ASTSummary is a token-efficient structural summary of a source file
// sent to the LLM for deep architectural analysis.
type ASTSummary struct {
	File    string     `json:"file"`
	Package string     `json:"package,omitempty"`
	Imports []string   `json:"imports,omitempty"`
	Types   []TypeNode `json:"types,omitempty"`
	Funcs   []FuncNode `json:"funcs,omitempty"`
}

// TypeNode represents a struct or class.
type TypeNode struct {
	Name    string       `json:"name"`
	Kind    string       `json:"kind"` // "struct" | "class"
	Line    int          `json:"line"`
	Fields  []FieldNode  `json:"fields,omitempty"`
	Methods []MethodNode `json:"methods,omitempty"`
}

// FieldNode is a named field of a struct or class.
type FieldNode struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// MethodNode represents a method attached to a type.
type MethodNode struct {
	Name       string   `json:"name"`
	Line       int      `json:"line"`
	Params     []string `json:"params,omitempty"`
	Complexity int      `json:"complexity,omitempty"`
	Lines      int      `json:"lines,omitempty"`
	Calls      []string `json:"calls,omitempty"`
}

// FuncNode represents a standalone (non-method) function.
type FuncNode struct {
	Name       string   `json:"name"`
	Line       int      `json:"line"`
	Params     []string `json:"params,omitempty"`
	Complexity int      `json:"complexity,omitempty"`
	Lines      int      `json:"lines,omitempty"`
	Calls      []string `json:"calls,omitempty"`
}

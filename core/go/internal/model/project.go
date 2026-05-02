package model

type ProjectModel struct {
	Root         string           `json:"root"`
	Files        []FileModel      `json:"files"`
	Symbols      []SymbolModel    `json:"symbols,omitempty"`
	Dependencies []DependencyEdge `json:"dependencies"`
	Findings     []Finding        `json:"findings,omitempty"`
}

package model

type ProjectModel struct {
	Root         string           `json:"root"`
	Files        []FileModel      `json:"files"`
	Dependencies []DependencyEdge `json:"dependencies"`
	Findings     []Finding        `json:"findings,omitempty"`
}

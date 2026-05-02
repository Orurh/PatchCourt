package model

type DependencyKind string

const (
	DependencyKindInclude DependencyKind = "include"
	DependencyKindImport  DependencyKind = "import"
)

type DependencyUsage string

const (
	DependencyUsageUnknown DependencyUsage = "unknown"
	DependencyUsageUsed    DependencyUsage = "used"
	DependencyUsageUnused  DependencyUsage = "unused"
	DependencyUsageMaybe   DependencyUsage = "maybe"
)

type ResolutionSource string

const (
	ResolutionSourceNone      ResolutionSource = "none"
	ResolutionSourceConfig    ResolutionSource = "config"
	ResolutionSourceHeuristic ResolutionSource = "heuristic"
)

type ResolutionConfidence string

const (
	ResolutionConfidenceNone   ResolutionConfidence = "none"
	ResolutionConfidenceLow    ResolutionConfidence = "low"
	ResolutionConfidenceMedium ResolutionConfidence = "medium"
	ResolutionConfidenceHigh   ResolutionConfidence = "high"
)

type DependencyEdge struct {
	FromFile             string               `json:"from_file"`
	ToFile               string               `json:"to_file,omitempty"`
	Target               string               `json:"target"`
	FromLayer            string               `json:"from_layer,omitempty"`
	ToLayer              string               `json:"to_layer,omitempty"`
	Kind                 DependencyKind       `json:"kind"`
	Usage                DependencyUsage      `json:"usage,omitempty"`
	Resolved             bool                 `json:"resolved"`
	External             bool                 `json:"external"`
	ResolutionSource     ResolutionSource     `json:"resolution_source"`
	ResolutionConfidence ResolutionConfidence `json:"resolution_confidence"`
	Ambiguous            bool                 `json:"ambiguous,omitempty"`
	Candidates           []string             `json:"candidates,omitempty"`
}

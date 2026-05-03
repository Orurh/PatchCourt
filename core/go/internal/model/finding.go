package model

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

type FindingKind string

const (
	FindingKindUnknown         FindingKind = ""
	FindingKindFactDiagnostic  FindingKind = "fact_diagnostic"
	FindingKindDiscoveryHint   FindingKind = "discovery_hint"
	FindingKindPolicyViolation FindingKind = "policy_violation"
	FindingKindReviewChange    FindingKind = "review_change"
	FindingKindRefactorHint    FindingKind = "refactor_hint"
)

type Finding struct {
	ID         string      `json:"id"`
	Kind       FindingKind `json:"kind,omitempty"`
	Severity   Severity    `json:"severity"`
	Title      string      `json:"title"`
	Evidence   []Evidence  `json:"evidence,omitempty"`
	Risk       string      `json:"risk,omitempty"`
	Suggestion string      `json:"suggestion,omitempty"`
	Confidence Confidence  `json:"confidence"`
}

type Evidence struct {
	File      string `json:"file"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
	Snippet   string `json:"snippet,omitempty"`
	Message   string `json:"message,omitempty"`
	FromLayer string `json:"from_layer,omitempty"`
	ToLayer   string `json:"to_layer,omitempty"`
	FromFile  string `json:"from_file,omitempty"`
	ToFile    string `json:"to_file,omitempty"`
}

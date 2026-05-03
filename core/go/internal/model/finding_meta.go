package model

func HumanFindingKind(kind FindingKind) string {
	switch kind {
	case FindingKindPolicyViolation:
		return "policy violation"
	case FindingKindDiscoveryHint:
		return "discovery hint"
	case FindingKindRefactorHint:
		return "refactor hint"
	case FindingKindReviewChange:
		return "review change"
	case FindingKindFactDiagnostic:
		return "fact diagnostic"
	default:
		return "finding"
	}
}

func SeverityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 100
	case SeverityHigh:
		return 80
	case SeverityMedium:
		return 50
	case SeverityLow:
		return 10
	default:
		return 0
	}
}

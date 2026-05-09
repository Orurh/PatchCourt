package runtime

import "regexp"

var (
	rawPointerFromGetRE   = regexp.MustCompile(`(?:^|[=\s;{(])(?:auto|[A-Za-z_][A-Za-z0-9_:<>]*)\s*\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*[^;]*\.get\s*\(`)
	lambdaCaptureRE       = regexp.MustCompile(`\[([^\]]+)\]`)
	structuredBindingRE   = regexp.MustCompile(`\b(?:auto|const\s+auto)\s*&?\s*\[([^\]]+)\]`)
	localVariableShadowRE = regexp.MustCompile(`\b(?:auto|const\s+auto|[A-Za-z_][A-Za-z0-9_:<>]*)\s*[*&]?\s*([A-Za-z_][A-Za-z0-9_]*)\b`)
	loopKeywordRE         = regexp.MustCompile(`\b(while|for|do)\b`)
)

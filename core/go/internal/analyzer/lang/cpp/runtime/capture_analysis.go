package runtime

import (
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

func analyzeCaptures(
	file model.FileModel,
	lineNumber int,
	original string,
	lines []string,
	index int,
	captures []string,
	context runtimeContext,
	rawPointers map[string]rawPointerCandidate,
	builders map[string]*findingBuilder,
) {
	for _, capture := range captures {
		switch capture {
		case "this":
			if !isReportableThisCaptureContext(context.Kind) {
				continue
			}

			emitRuntimeSite(
				newThisCaptureRuntimeSite(file, lineNumber, lines, index, captures, context),
				builders,
			)

		default:
			candidate, ok := rawPointers[capture]
			if !ok || !isReportableRawPointerCaptureContext(context.Kind) {
				continue
			}

			if lineNumber-candidate.line > rawPointerCandidateMaxDistance {
				continue
			}

			emitRuntimeSite(
				newRawPointerCaptureRuntimeSite(file, lineNumber, lines, index, captures, context, candidate),
				builders,
			)
		}
	}
}
func rawPointerNamesFromLine(line string) []string {
	matches := rawPointerFromGetRE.FindAllStringSubmatch(line, -1)
	result := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		result = append(result, match[1])
	}

	return result
}
func expireRawPointerCandidates(rawPointers map[string]rawPointerCandidate, lineNumber int) {
	for name, candidate := range rawPointers {
		if lineNumber-candidate.line > rawPointerCandidateMaxDistance {
			delete(rawPointers, name)
		}
	}
}
func removeShadowedRawPointers(rawPointers map[string]rawPointerCandidate, line string) {
	for _, name := range structuredBindingNamesFromLine(line) {
		delete(rawPointers, name)
	}

	for _, name := range localVariableNamesFromLine(line) {
		if _, ok := rawPointers[name]; ok && len(rawPointerNamesFromLine(line)) == 0 {
			delete(rawPointers, name)
		}
	}
}
func structuredBindingNamesFromLine(line string) []string {
	matches := structuredBindingRE.FindAllStringSubmatch(line, -1)
	result := make([]string, 0)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		for _, part := range strings.Split(match[1], ",") {
			name := strings.TrimSpace(part)
			name = strings.TrimPrefix(name, "&")
			name = strings.TrimPrefix(name, "*")
			name = strings.TrimSpace(name)
			if name != "" {
				result = append(result, name)
			}
		}
	}

	return result
}
func localVariableNamesFromLine(line string) []string {
	matches := localVariableShadowRE.FindAllStringSubmatch(line, -1)
	result := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		name := match[1]
		if name == "auto" || name == "const" || name == "return" || name == "if" || name == "for" || name == "while" {
			continue
		}

		result = append(result, name)
	}

	return result
}
func lambdaCaptures(line string) []string {
	matches := lambdaCaptureRE.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return nil
	}

	var result []string

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		captureText := line[match[2]:match[3]]
		afterCapture := line[match[1]:]

		if !looksLikeLambdaAfterCapture(afterCapture) {
			continue
		}

		parts := strings.Split(captureText, ",")
		for _, part := range parts {
			capture := normalizeLambdaCapture(part)
			if capture == "" {
				continue
			}

			result = append(result, capture)
		}
	}

	return result
}
func looksLikeLambdaAfterCapture(after string) bool {
	after = strings.TrimSpace(after)
	if after == "" {
		return false
	}

	return strings.HasPrefix(after, "(") ||
		strings.HasPrefix(after, "{") ||
		strings.HasPrefix(after, "<") ||
		strings.HasPrefix(after, "mutable") ||
		strings.HasPrefix(after, "noexcept") ||
		strings.HasPrefix(after, "->")
}
func normalizeLambdaCapture(part string) string {
	capture := strings.TrimSpace(part)
	capture = strings.TrimPrefix(capture, "&")
	capture = strings.TrimPrefix(capture, "=")
	capture = strings.TrimPrefix(capture, "*")
	capture = strings.TrimSpace(capture)

	if capture == "" {
		return ""
	}

	if strings.Contains(capture, "=") {
		capture = strings.TrimSpace(strings.SplitN(capture, "=", 2)[0])
	}

	return capture
}

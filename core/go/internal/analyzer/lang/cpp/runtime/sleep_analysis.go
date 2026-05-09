package runtime

import (
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

func analyzeShutdownSleep(
	file model.FileModel,
	lineNumber int,
	original string,
	lines []string,
	index int,
	builders map[string]*findingBuilder,
) {
	line := stripLineComment(original)
	if !isSleepLine(line) {
		return
	}

	if isDetachedDelayedCallbackAround(lines, index) {
		addEvidence(builders[findingDetachedDelayedCallback], model.Evidence{
			File:      file.Path,
			LineStart: lineNumber,
			Snippet:   evidenceSnippet(lines, index, 4, 5),
			Message:   "detached thread sleeps before invoking a shutdown/callback function",
		})
		return
	}

	if isPollingSleepAround(lines, index) {
		addEvidence(builders[findingShutdownPolling], model.Evidence{
			File:      file.Path,
			LineStart: lineNumber,
			Snippet:   evidenceSnippet(lines, index, 4, 5),
			Message:   "sleep/polling-like wait appears inside a shutdown/callback-draining loop",
		})
	}
}
func isSleepLine(line string) bool {
	lower := strings.ToLower(line)

	return strings.Contains(lower, "sleep_for") ||
		strings.Contains(lower, "usleep") ||
		strings.Contains(lower, "sleep(")
}
func isDetachedDelayedCallbackAround(lines []string, index int) bool {
	text := strings.ToLower(rawWindowText(lines, index, 4, 6))

	if !strings.Contains(text, "std::thread") || !strings.Contains(text, ".detach(") {
		return false
	}

	if !strings.Contains(text, "callback") && !strings.Contains(text, "shutdown") {
		return false
	}

	return strings.Contains(text, "sleep_for") ||
		strings.Contains(text, "usleep") ||
		strings.Contains(text, "sleep(")
}
func isPollingSleepAround(lines []string, index int) bool {
	text := strings.ToLower(rawWindowText(lines, index, 4, 4))

	if !loopKeywordRE.MatchString(text) {
		return false
	}

	return strings.Contains(text, "pending") ||
		strings.Contains(text, "callback") ||
		strings.Contains(text, "shutdown") ||
		strings.Contains(text, "disconnect") ||
		strings.Contains(text, "stop") ||
		strings.Contains(text, "join") ||
		strings.Contains(text, ".load()")
}

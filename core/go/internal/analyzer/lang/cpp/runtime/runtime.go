package runtime

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

const (
	findingRawPointerCapture       = "cpp.async.raw_pointer_capture"
	findingThisCapture             = "cpp.async.this_capture"
	findingShutdownPolling         = "cpp.shutdown.sleep_polling"
	findingDetachedDelayedCallback = "cpp.shutdown.detached_delayed_callback"
)

var (
	rawPointerFromGetRE = regexp.MustCompile(`(?:^|[=\s;{(])(?:auto|[A-Za-z_][A-Za-z0-9_:<>]*)\s*\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*[^;]*\.get\s*\(`)
	lambdaCaptureRE     = regexp.MustCompile(`\[([^\]]+)\]`)
)

type rawPointerCandidate struct {
	name string
	line int
}

type delayedCallbackCandidate struct {
	line    int
	snippet string
}

type findingBuilder struct {
	finding model.Finding
}

func Analyze(root string, project *model.ProjectModel) []model.Finding {
	if project == nil {
		return nil
	}

	builders := map[string]*findingBuilder{
		findingRawPointerCapture: {
			finding: model.Finding{
				ID:         findingRawPointerCapture,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityHigh,
				Title:      "Raw pointer captured into async task",
				Risk:       "Object lifetime is not tied to async task lifetime. A mutex may protect lookup, but not lifetime after the task is scheduled.",
				Suggestion: "Verify the lifetime contract. Prefer shared_ptr/weak_ptr guards, cancellation tokens, or owner-bound async execution.",
				Confidence: model.ConfidenceHigh,
			},
		},
		findingThisCapture: {
			finding: model.Finding{
				ID:         findingThisCapture,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityHigh,
				Title:      "`this` captured into async callback",
				Risk:       "Callback may outlive the owning object unless object lifetime is guarded by shared_ptr/weak_ptr, cancellation, strand ownership, or another visible lifetime contract.",
				Suggestion: "Review what guarantees that the owning object outlives the callback. Consider weak_ptr guard or explicit cancellation/lifecycle ownership.",
				Confidence: model.ConfidenceMedium,
			},
		},
		findingShutdownPolling: {
			finding: model.Finding{
				ID:         findingShutdownPolling,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityMedium,
				Title:      "Shutdown or callback draining uses sleep/polling",
				Risk:       "Shutdown order may depend on pending callbacks and cross-thread ownership. Sleep/polling loops do not prove that async work is safely drained.",
				Suggestion: "Review cancellation, callback completion, and ownership contracts. Prefer explicit completion aggregation, condition_variable, future/promise, or structured async shutdown.",
				Confidence: model.ConfidenceMedium,
			},
		},
		findingDetachedDelayedCallback: {
			finding: model.Finding{
				ID:         findingDetachedDelayedCallback,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityMedium,
				Title:      "Detached delayed shutdown callback",
				Risk:       "A detached thread delays and invokes a shutdown callback without visible join, cancellation, or owner lifetime tracking.",
				Suggestion: "Prefer structured shutdown scheduling owned by the server/event loop, or make the delayed shutdown worker joinable/cancellable with explicit lifetime ownership.",
				Confidence: model.ConfidenceMedium,
			},
		},
	}

	for _, file := range project.Files {
		if !isReviewableCPPFile(file) {
			continue
		}

		analyzeFile(root, file, builders)
	}

	findings := make([]model.Finding, 0, len(builders))
	for _, builder := range builders {
		if len(builder.finding.Evidence) == 0 {
			continue
		}

		sortEvidence(builder.finding.Evidence)
		findings = append(findings, builder.finding)
	}

	sort.Slice(findings, func(i int, j int) bool {
		return findings[i].ID < findings[j].ID
	})

	return findings
}

func isReviewableCPPFile(file model.FileModel) bool {
	if file.Language != model.LanguageCPP {
		return false
	}

	if file.Role == model.FileRoleGenerated || file.Role == model.FileRoleExternal {
		return false
	}

	return file.Kind == model.FileKindSource || file.Kind == model.FileKindHeader
}

func analyzeFile(root string, file model.FileModel, builders map[string]*findingBuilder) {
	path := filepath.Join(root, filepath.FromSlash(file.Path))

	handle, err := os.Open(path)
	if err != nil {
		return
	}
	defer handle.Close()

	rawPointers := make(map[string]rawPointerCandidate)

	scanner := bufio.NewScanner(handle)
	lineNumber := 0

	pollingContextUntilLine := 0
	detachedThreadContextUntilLine := 0
	var delayedSleep *delayedCallbackCandidate

	for scanner.Scan() {
		lineNumber++

		original := scanner.Text()
		line := stripLineComment(original)

		if isPollingLoopContextLine(line) {
			pollingContextUntilLine = lineNumber + 5
		}

		if isDetachedThreadContextLine(line) {
			detachedThreadContextUntilLine = lineNumber + 10
			delayedSleep = nil
		}

		for _, name := range rawPointerNamesFromLine(line) {
			rawPointers[name] = rawPointerCandidate{
				name: name,
				line: lineNumber,
			}
		}

		captures := lambdaCaptures(line)
		if len(captures) > 0 && isAsyncLookingLine(line) {
			for _, capture := range captures {
				if capture == "this" {
					addEvidence(builders[findingThisCapture], model.Evidence{
						File:      file.Path,
						LineStart: lineNumber,
						Snippet:   strings.TrimSpace(original),
						Message:   "`this` is captured in an async-looking callback/task",
					})
				}

				candidate, ok := rawPointers[capture]
				if ok {
					addEvidence(builders[findingRawPointerCapture], model.Evidence{
						File:      file.Path,
						LineStart: lineNumber,
						Snippet:   strings.TrimSpace(original),
						Message: fmt.Sprintf(
							"raw pointer %q, declared at line %d, is captured in an async-looking callback/task",
							candidate.name,
							candidate.line,
						),
					})
				}
			}
		}

		if isPollingSleepLine(line) {
			inDetachedThreadContext := lineNumber <= detachedThreadContextUntilLine

			if inDetachedThreadContext {
				delayedSleep = &delayedCallbackCandidate{
					line:    lineNumber,
					snippet: strings.TrimSpace(original),
				}
			}

			if !inDetachedThreadContext && lineNumber <= pollingContextUntilLine {
				addEvidence(builders[findingShutdownPolling], model.Evidence{
					File:      file.Path,
					LineStart: lineNumber,
					Snippet:   strings.TrimSpace(original),
					Message:   "sleep/polling-like wait appears inside a shutdown/callback-draining loop",
				})
			}
		}

		if delayedSleep != nil && lineNumber <= detachedThreadContextUntilLine && isDetachedCallbackInvocationLine(line) {
			addEvidence(builders[findingDetachedDelayedCallback], model.Evidence{
				File:      file.Path,
				LineStart: delayedSleep.line,
				Snippet:   delayedSleep.snippet,
				Message:   "detached thread sleeps before invoking a shutdown/callback function",
			})
			delayedSleep = nil
		}

		if lineNumber <= detachedThreadContextUntilLine && isDetachLine(line) && delayedSleep != nil {
			addEvidence(builders[findingDetachedDelayedCallback], model.Evidence{
				File:      file.Path,
				LineStart: delayedSleep.line,
				Snippet:   delayedSleep.snippet,
				Message:   "detached thread sleeps before shutdown/callback completion",
			})
			delayedSleep = nil
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

func lambdaCaptures(line string) []string {
	match := lambdaCaptureRE.FindStringSubmatch(line)
	if len(match) != 2 {
		return nil
	}

	parts := strings.Split(match[1], ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		capture := strings.TrimSpace(part)
		capture = strings.TrimPrefix(capture, "&")
		capture = strings.TrimPrefix(capture, "=")
		capture = strings.TrimPrefix(capture, "*")
		capture = strings.TrimSpace(capture)

		if capture == "" {
			continue
		}

		result = append(result, capture)
	}

	return result
}

func isAsyncLookingLine(line string) bool {
	line = strings.ToLower(line)

	asyncMarkers := []string{
		"boost::asio::post",
		"asio::post",
		"post(",
		"dispatch(",
		"defer(",
		"async_",
		"async.",
		"callback",
		"set_callback",
		"sethandler",
		"set_handler",
		"timer",
		"thread_pool",
	}

	for _, marker := range asyncMarkers {
		if strings.Contains(line, marker) {
			return true
		}
	}

	return false
}

func isPollingLoopContextLine(line string) bool {
	lower := strings.ToLower(line)

	if !strings.Contains(lower, "while") &&
		!strings.Contains(lower, "for") &&
		!strings.Contains(lower, "do") {
		return false
	}

	return isShutdownContextLine(lower) ||
		strings.Contains(lower, "pending") ||
		strings.Contains(lower, "load()") ||
		strings.Contains(lower, "joinable")
}

func isPollingSleepLine(line string) bool {
	lower := strings.ToLower(line)

	return strings.Contains(lower, "sleep_for") ||
		strings.Contains(lower, "usleep") ||
		strings.Contains(lower, "sleep(")
}

func isShutdownContextLine(line string) bool {
	lower := strings.ToLower(line)

	return strings.Contains(lower, "shutdown") ||
		strings.Contains(lower, "stop") ||
		strings.Contains(lower, "disconnect") ||
		strings.Contains(lower, "pending") ||
		strings.Contains(lower, "callback") ||
		strings.Contains(lower, "join")
}

func isDetachedThreadContextLine(line string) bool {
	lower := strings.ToLower(line)

	if !strings.Contains(lower, "std::thread") {
		return false
	}

	return strings.Contains(lower, "shutdown") ||
		strings.Contains(lower, "callback")
}

func isDetachedCallbackInvocationLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))

	return strings.Contains(lower, "callback(") ||
		strings.Contains(lower, "shutdown_callback")
}

func isDetachLine(line string) bool {
	return strings.Contains(strings.ToLower(line), ".detach(") ||
		strings.Contains(strings.ToLower(line), ").detach")
}

func addEvidence(builder *findingBuilder, evidence model.Evidence) {
	if builder == nil {
		return
	}

	builder.finding.Evidence = append(builder.finding.Evidence, evidence)
}

func stripLineComment(line string) string {
	index := strings.Index(line, "//")
	if index < 0 {
		return line
	}

	return line[:index]
}

func sortEvidence(items []model.Evidence) {
	sort.SliceStable(items, func(i int, j int) bool {
		if items[i].File != items[j].File {
			return items[i].File < items[j].File
		}

		if items[i].LineStart != items[j].LineStart {
			return items[i].LineStart < items[j].LineStart
		}

		return items[i].Message < items[j].Message
	})
}

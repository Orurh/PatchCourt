package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

const (
	findingRawPointerAsyncCapture    = "cpp.lifetime.raw_pointer_async_capture"
	findingThisCaptureAsync          = "cpp.lifetime.this_capture_async"
	findingThisCaptureStoredCallback = "cpp.lifetime.this_capture_stored_callback"
	findingThisCaptureThread         = "cpp.lifetime.this_capture_thread"
	findingShutdownPolling           = "cpp.shutdown.sleep_polling"
	findingDetachedDelayedCallback   = "cpp.shutdown.detached_delayed_callback"
)

var (
	rawPointerFromGetRE   = regexp.MustCompile(`(?:^|[=\s;{(])(?:auto|[A-Za-z_][A-Za-z0-9_:<>]*)\s*\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*[^;]*\.get\s*\(`)
	lambdaCaptureRE       = regexp.MustCompile(`\[([^\]]+)\]`)
	structuredBindingRE   = regexp.MustCompile(`\b(?:auto|const\s+auto)\s*&?\s*\[([^\]]+)\]`)
	localVariableShadowRE = regexp.MustCompile(`\b(?:auto|const\s+auto|[A-Za-z_][A-Za-z0-9_:<>]*)\s*[*&]?\s*([A-Za-z_][A-Za-z0-9_]*)\b`)
	loopKeywordRE         = regexp.MustCompile(`\b(while|for|do)\b`)
)

const rawPointerCandidateMaxDistance = 20

type rawPointerCandidate struct {
	name string
	line int
}

type findingBuilder struct {
	finding model.Finding
}

func Analyze(root string, project *model.ProjectModel) []model.Finding {
	if project == nil {
		return nil
	}

	builders := runtimeFindingBuilders()

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

func runtimeFindingBuilders() map[string]*findingBuilder {
	return map[string]*findingBuilder{
		findingRawPointerAsyncCapture: {
			finding: model.Finding{
				ID:         findingRawPointerAsyncCapture,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityHigh,
				Title:      "Raw pointer captured into deferred async/thread task",
				Risk:       "Object lifetime is not visibly tied to the deferred task lifetime. A mutex may protect lookup, but not lifetime after the task is scheduled.",
				Suggestion: "Verify the lifetime contract. Prefer shared_ptr/weak_ptr guards, cancellation tokens, or owner-bound async execution.",
				Confidence: model.ConfidenceHigh,
			},
		},
		findingThisCaptureAsync: {
			finding: model.Finding{
				ID:         findingThisCaptureAsync,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityHigh,
				Title:      "`this` captured into async callback",
				Risk:       "Callback may outlive the owning object unless object lifetime is guarded by shared_ptr/weak_ptr, cancellation, strand ownership, or another visible lifetime contract.",
				Suggestion: "Review what guarantees that the owning object outlives the callback. Consider weak_ptr guard or explicit cancellation/lifecycle ownership.",
				Confidence: model.ConfidenceMedium,
			},
		},
		findingThisCaptureStoredCallback: {
			finding: model.Finding{
				ID:         findingThisCaptureStoredCallback,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityMedium,
				Title:      "`this` captured into stored callback",
				Risk:       "Callback appears to be stored in another object. If that object can outlive the owner, the callback may call a destroyed object.",
				Suggestion: "Verify callback ownership and reset/cancellation in shutdown/destructor. Prefer weak_ptr guard or explicit callback lifetime ownership.",
				Confidence: model.ConfidenceMedium,
			},
		},
		findingThisCaptureThread: {
			finding: model.Finding{
				ID:         findingThisCaptureThread,
				Kind:       model.FindingKindRuntimeRisk,
				Severity:   model.SeverityHigh,
				Title:      "`this` captured into thread callback",
				Risk:       "Thread callback may outlive the owning object unless the thread is joined/cancelled before destruction.",
				Suggestion: "Verify thread ownership and shutdown ordering. Prefer joinable owned threads, cancellation tokens, or shared/weak lifetime guards.",
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
				Risk:       "A detached thread delays and invokes a shutdown/callback function without visible join, cancellation, or owner lifetime tracking.",
				Suggestion: "Prefer structured shutdown scheduling owned by the server/event loop, or make the delayed shutdown worker joinable/cancellable with explicit lifetime ownership.",
				Confidence: model.ConfidenceMedium,
			},
		},
	}
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

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := splitLines(string(data))
	rawPointers := make(map[string]rawPointerCandidate)

	for i, original := range lines {
		lineNumber := i + 1
		line := stripLineComment(original)

		expireRawPointerCandidates(rawPointers, lineNumber)
		removeShadowedRawPointers(rawPointers, line)

		for _, name := range rawPointerNamesFromLine(line) {
			rawPointers[name] = rawPointerCandidate{
				name: name,
				line: lineNumber,
			}
		}

		captures := lambdaCaptures(line)
		if len(captures) > 0 {
			context := classifyRuntimeContext(buildLineWindow(lines, i))
			analyzeCaptures(file, lineNumber, original, lines, i, captures, context, rawPointers, builders)
		}

		analyzeShutdownSleep(file, lineNumber, original, lines, i, builders)
	}
}

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
		switch  capture{
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

func emitRuntimeSite(site RuntimeSite, builders map[string]*findingBuilder) {
	if site.Score <= 0 || site.FindingID == "" || site.Severity == "" {
		return
	}

	builder := builders[site.FindingID]
	if builder == nil {
		return
	}

	if len(builder.finding.Evidence) == 0 {
		builder.finding.Severity = site.Severity
		builder.finding.Confidence = site.Confidence
	} else {
		builder.finding.Severity = maxSeverity(builder.finding.Severity, site.Severity)
		builder.finding.Confidence = maxConfidence(builder.finding.Confidence, site.Confidence)
	}

	addEvidence(builder, model.Evidence{
		File:      site.File,
		LineStart: site.Line,
		Snippet:   site.Snippet,
		Message:   site.Message,
	})
}

func maxSeverity(left model.Severity, right model.Severity) model.Severity {
	if model.SeverityRank(right) > model.SeverityRank(left) {
		return right
	}

	return left
}

func maxConfidence(left model.Confidence, right model.Confidence) model.Confidence {
	if runtimeConfidenceRank(right) > runtimeConfidenceRank(left) {
		return right
	}

	return left
}

func runtimeConfidenceRank(confidence model.Confidence) int {
	switch confidence {
	case model.ConfidenceHigh:
		return 3
	case model.ConfidenceMedium:
		return 2
	case model.ConfidenceLow:
		return 1
	default:
		return 0
	}
}

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

func splitLines(data string) []string {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	data = strings.ReplaceAll(data, "\r", "\n")
	return strings.Split(data, "\n")
}

func buildLineWindow(lines []string, index int) lineWindow {
	beforeStart := index - 3
	if beforeStart < 0 {
		beforeStart = 0
	}

	for beforeStart < index {
		trimmed := strings.TrimSpace(stripLineComment(lines[beforeStart]))
		if trimmed == "" || isStatementBoundary(trimmed) {
			beforeStart++
			continue
		}

		break
	}

	afterEnd := index + 1
	for afterEnd < len(lines) && afterEnd < index+7 {
		afterEnd++

		trimmed := strings.TrimSpace(stripLineComment(lines[afterEnd-1]))
		if isStatementBoundary(trimmed) {
			break
		}
	}

	before := append([]string(nil), lines[beforeStart:index]...)
	after := append([]string(nil), lines[index+1:afterEnd]...)

	return lineWindow{
		Before: before,
		Line:   lines[index],
		After:  after,
	}
}

func isStatementBoundary(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return true
	}

	return strings.HasSuffix(line, ";") ||
		strings.HasSuffix(line, "};") ||
		strings.HasSuffix(line, "});") ||
		strings.HasSuffix(line, "})") ||
		line == "}"
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

func rawWindowText(lines []string, index int, before int, after int) string {
	start := index - before
	if start < 0 {
		start = 0
	}

	end := index + after + 1
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for _, line := range lines[start:end] {
		b.WriteString(stripLineComment(line))
		b.WriteByte('\n')
	}

	return b.String()
}

func evidenceSnippet(lines []string, index int, before int, after int) string {
	start := index - before
	if start < 0 {
		start = 0
	}

	end := index + after + 1
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		prefix := "  "
		if i == index {
			prefix = "> "
		}

		fmt.Fprintf(&b, "%s%4d | %s\n", prefix, i+1, strings.TrimRight(lines[i], " \t"))
	}

	return strings.TrimRight(b.String(), "\n")
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

package runtime

import (
	"fmt"
	"strings"

	"github.com/orurh/patchcourt/internal/model"
)

type RuntimeGuardKind string

const (
	RuntimeGuardSharedSelf    RuntimeGuardKind = "shared_self"
	RuntimeGuardWeakLock      RuntimeGuardKind = "weak_lock"
	RuntimeGuardCancelOrJoin  RuntimeGuardKind = "cancel_or_join"
	RuntimeGuardCallbackReset RuntimeGuardKind = "callback_reset"
)

type RuntimeGuard struct {
	Kind       RuntimeGuardKind `json:"kind"`
	Marker     string           `json:"marker,omitempty"`
	ScoreDelta int              `json:"score_delta"`
}

type RuntimeSite struct {
	File    string
	Line    int
	Snippet string

	CapturesThis   bool
	CapturesRawPtr bool
	RawPtrName     string

	Context RuntimeContext
	Guards  []RuntimeGuard

	Score      int
	FindingID  string
	Severity   model.Severity
	Confidence model.Confidence
	Message    string
}

func newThisCaptureRuntimeSite(file model.FileModel, lineNumber int, lines []string, index int, captures []string, context runtimeContext) RuntimeSite {
	site := RuntimeSite{
		File:         file.Path,
		Line:         lineNumber,
		Snippet:      evidenceSnippet(lines, index, 3, 5),
		CapturesThis: true,
		Context:      context,
		Guards:       detectRuntimeGuards(captures, lines, index),
		FindingID:    findingIDForThisCaptureContext(context.Kind),
	}

	site.Score = scoreRuntimeSite(site)
	site.Severity = severityForRuntimeScore(site.Score)
	site.Confidence = confidenceForRuntimeSite(site, model.ConfidenceMedium)
	site.Message = runtimeSiteMessage(site)

	return site
}

func newRawPointerCaptureRuntimeSite(file model.FileModel, lineNumber int, lines []string, index int, captures []string, context runtimeContext, candidate rawPointerCandidate) RuntimeSite {
	site := RuntimeSite{
		File:           file.Path,
		Line:           lineNumber,
		Snippet:        evidenceSnippet(lines, index, 3, 5),
		CapturesRawPtr: true,
		RawPtrName:     candidate.name,
		Context:        context,
		Guards:         detectRuntimeGuards(captures, lines, index),
		FindingID:      findingRawPointerAsyncCapture,
	}

	site.Score = scoreRuntimeSite(site)
	site.Severity = severityForRuntimeScore(site.Score)
	site.Confidence = confidenceForRuntimeSite(site, model.ConfidenceHigh)
	site.Message = fmt.Sprintf(
		"raw pointer %q, declared at line %d, is captured in %s context: %s; score=%d%s",
		candidate.name,
		candidate.line,
		context.Kind,
		context.Reason,
		site.Score,
		guardMessageSuffix(site.Guards),
	)

	return site
}

func scoreRuntimeSite(site RuntimeSite) int {
	score := ContextBaseScore(site.Context.Kind)

	if site.CapturesRawPtr && isReportableThisCaptureContext(site.Context.Kind) {
		if score < 5 {
			score = 5
		}
	}

	for _, guard := range site.Guards {
		switch guard.Kind {
		case RuntimeGuardSharedSelf:
			score -= 3
		case RuntimeGuardWeakLock:
			score -= 3
		case RuntimeGuardCancelOrJoin:
			score -= 1
		case RuntimeGuardCallbackReset:
			score -= 1
		}
	}

	if score < 0 {
		return 0
	}

	return score
}

func severityForRuntimeScore(score int) model.Severity {
	switch {
	case score >= 5:
		return model.SeverityHigh
	case score >= 3:
		return model.SeverityMedium
	case score >= 1:
		return model.SeverityLow
	default:
		return ""
	}
}

func confidenceForRuntimeSite(site RuntimeSite, base model.Confidence) model.Confidence {
	if site.Score <= 0 {
		return ""
	}

	if hasRuntimeGuard(site.Guards, RuntimeGuardWeakLock) {
		return model.ConfidenceLow
	}

	if hasRuntimeGuard(site.Guards, RuntimeGuardSharedSelf) || hasRuntimeGuard(site.Guards, RuntimeGuardCancelOrJoin) {
		return model.ConfidenceMedium
	}

	return base
}

func runtimeSiteMessage(site RuntimeSite) string {
	subject := "runtime callback site"
	if site.CapturesThis {
		subject = "`this` is captured"
	}
	if site.CapturesRawPtr {
		subject = fmt.Sprintf("raw pointer %q is captured", site.RawPtrName)
	}

	return fmt.Sprintf(
		"%s in %s context: %s; score=%d%s",
		subject,
		site.Context.Kind,
		site.Context.Reason,
		site.Score,
		guardMessageSuffix(site.Guards),
	)
}

func detectRuntimeGuards(captures []string, lines []string, index int) []RuntimeGuard {
	text := strings.ToLower(rawWindowText(lines, index, 3, 6))
	guards := make([]RuntimeGuard, 0, 4)

	if capturesContain(captures, "self") || strings.Contains(text, "shared_from_this") || strings.Contains(text, "enable_shared_from_this") || strings.Contains(text, "self =") {
		guards = append(guards, RuntimeGuard{Kind: RuntimeGuardSharedSelf, Marker: "shared/self capture", ScoreDelta: -3})
	}

	if strings.Contains(text, "weak_from_this") || strings.Contains(text, "std::weak_ptr") || strings.Contains(text, "weak_ptr") || strings.Contains(text, "self.lock()") || strings.Contains(text, ".lock()") {
		guards = append(guards, RuntimeGuard{Kind: RuntimeGuardWeakLock, Marker: "weak_ptr lock", ScoreDelta: -3})
	}

	if strings.Contains(text, "timer.cancel") || strings.Contains(text, ".cancel(") || strings.Contains(text, "cancel()") || strings.Contains(text, "stop()") || strings.Contains(text, ".stop(") || strings.Contains(text, "join()") || strings.Contains(text, ".join(") || strings.Contains(text, "work_guard_.reset()") {
		guards = append(guards, RuntimeGuard{Kind: RuntimeGuardCancelOrJoin, Marker: "cancel/stop/join/reset", ScoreDelta: -1})
	}

	if strings.Contains(text, "callback_ = nullptr") || strings.Contains(text, "callback_=nullptr") || strings.Contains(text, "handler_ = nullptr") || strings.Contains(text, "handler_=nullptr") || strings.Contains(text, "reset_callback") || strings.Contains(text, "clear_callback") {
		guards = append(guards, RuntimeGuard{Kind: RuntimeGuardCallbackReset, Marker: "callback reset", ScoreDelta: -1})
	}

	return guards
}

func capturesContain(captures []string, target string) bool {
	for _, capture := range captures {
		if capture == target {
			return true
		}
	}

	return false
}

func hasRuntimeGuard(guards []RuntimeGuard, kind RuntimeGuardKind) bool {
	for _, guard := range guards {
		if guard.Kind == kind {
			return true
		}
	}

	return false
}

func guardMessageSuffix(guards []RuntimeGuard) string {
	if len(guards) == 0 {
		return ""
	}

	parts := make([]string, 0, len(guards))
	for _, guard := range guards {
		if guard.Marker != "" {
			parts = append(parts, guard.Marker)
			continue
		}

		parts = append(parts, string(guard.Kind))
	}

	return "; visible guards: " + strings.Join(parts, ", ")
}

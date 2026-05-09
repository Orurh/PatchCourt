package runtime

import "github.com/orurh/patchcourt/internal/model"

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

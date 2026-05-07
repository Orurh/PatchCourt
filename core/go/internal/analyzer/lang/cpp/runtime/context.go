package runtime

import "strings"

type runtimeContextKind string

const (
	contextUnknown        runtimeContextKind = "unknown"
	contextSyncPredicate  runtimeContextKind = "sync_predicate"
	contextLocalAlgorithm runtimeContextKind = "local_algorithm"
	contextAsioPost       runtimeContextKind = "asio_post"
	contextAsioAsyncOp    runtimeContextKind = "asio_async_op"
	contextTimerCallback  runtimeContextKind = "timer_callback"
	contextStoredCallback runtimeContextKind = "stored_callback"
	contextThreadStart    runtimeContextKind = "thread_start"
	contextDetachedThread runtimeContextKind = "detached_thread"
)

type lineWindow struct {
	Before []string
	Line   string
	After  []string
}

type runtimeContext struct {
	Kind   runtimeContextKind
	Reason string
}

func classifyRuntimeContext(window lineWindow) runtimeContext {
	lineText := strings.ToLower(stripLineComment(window.Line))
	if context, ok := classifyRuntimeContextText(lineText); ok {
		return context
	}

	windowText := strings.ToLower(windowText(window))
	if context, ok := classifyRuntimeContextText(windowText); ok {
		return context
	}

	return runtimeContext{
		Kind:   contextUnknown,
		Reason: "lambda context is not recognized as deferred/stored/threaded",
	}
}

func classifyRuntimeContextText(text string) (runtimeContext, bool) {
	switch {
	case isSyncPredicateContext(text):
		return runtimeContext{
			Kind:   contextSyncPredicate,
			Reason: "lambda is used as a synchronous wait predicate",
		}, true

	case isLocalAlgorithmContext(text):
		return runtimeContext{
			Kind:   contextLocalAlgorithm,
			Reason: "lambda appears to be used by a synchronous standard algorithm",
		}, true

	case isAsioPostContext(text):
		return runtimeContext{
			Kind:   contextAsioPost,
			Reason: "lambda is scheduled through asio post/dispatch/defer",
		}, true

	case isTimerCallbackContext(text):
		return runtimeContext{
			Kind:   contextTimerCallback,
			Reason: "lambda is used as a timer callback",
		}, true

	case isAsioAsyncContext(text):
		return runtimeContext{
			Kind:   contextAsioAsyncOp,
			Reason: "lambda is used as an asio async operation callback",
		}, true

	case isStoredCallbackContext(text):
		return runtimeContext{
			Kind:   contextStoredCallback,
			Reason: "lambda appears to be stored as a callback/handler",
		}, true

	case isDetachedThreadContext(text):
		return runtimeContext{
			Kind:   contextDetachedThread,
			Reason: "lambda is passed to a detached std::thread",
		}, true

	case isThreadStartContext(text):
		return runtimeContext{
			Kind:   contextThreadStart,
			Reason: "lambda is passed to std::thread",
		}, true

	default:
		return runtimeContext{}, false
	}
}

func windowText(window lineWindow) string {
	var b strings.Builder

	for _, line := range window.Before {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(window.Line)
	b.WriteByte('\n')

	for _, line := range window.After {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String()
}

func isReportableThisCaptureContext(kind runtimeContextKind) bool {
	switch kind {
	case contextAsioPost,
		contextAsioAsyncOp,
		contextTimerCallback,
		contextStoredCallback,
		contextThreadStart,
		contextDetachedThread:
		return true
	default:
		return false
	}
}

func isReportableRawPointerCaptureContext(kind runtimeContextKind) bool {
	switch kind {
	case contextAsioPost,
		contextAsioAsyncOp,
		contextTimerCallback,
		contextThreadStart,
		contextDetachedThread:
		return true
	default:
		return false
	}
}

func findingIDForThisCaptureContext(kind runtimeContextKind) string {
	switch kind {
	case contextStoredCallback:
		return findingThisCaptureStoredCallback
	case contextThreadStart, contextDetachedThread:
		return findingThisCaptureThread
	default:
		return findingThisCaptureAsync
	}
}

func isSyncPredicateContext(text string) bool {
	return strings.Contains(text, ".wait_for(") ||
		strings.Contains(text, "wait_for(") ||
		strings.Contains(text, ".wait_until(") ||
		strings.Contains(text, "wait_until(") ||
		strings.Contains(text, ".wait(")
}

func isLocalAlgorithmContext(text string) bool {
	markers := []string{
		"std::sort(",
		"std::find_if(",
		"std::any_of(",
		"std::all_of(",
		"std::none_of(",
		"std::remove_if(",
		"std::transform(",
		"std::for_each(",
	}

	return containsAny(text, markers)
}

func isDetachedThreadContext(text string) bool {
	return strings.Contains(text, "std::thread") && strings.Contains(text, ".detach(")
}

func isThreadStartContext(text string) bool {
	return strings.Contains(text, "std::thread")
}

func isAsioPostContext(text string) bool {
	return strings.Contains(text, "boost::asio::post") ||
		strings.Contains(text, "asio::post") ||
		strings.Contains(text, "boost::asio::dispatch") ||
		strings.Contains(text, "asio::dispatch") ||
		strings.Contains(text, "boost::asio::defer") ||
		strings.Contains(text, "asio::defer")
}

func isTimerCallbackContext(text string) bool {
	return strings.Contains(text, "async_wait") ||
		strings.Contains(text, "steady_timer") ||
		strings.Contains(text, "deadline_timer")
}

func isAsioAsyncContext(text string) bool {
	markers := []string{
		"async_accept",
		"async_read",
		"async_write",
		"async_connect",
		"async_resolve",
		"async_receive",
		"async_send",
		"async_",
	}

	return containsAny(text, markers)
}

func isStoredCallbackContext(text string) bool {
	compact := strings.ReplaceAll(text, "_", "")
	compact = strings.ReplaceAll(compact, "-", "")

	return strings.Contains(compact, "set") && strings.Contains(compact, "callback(") ||
		strings.Contains(compact, "set") && strings.Contains(compact, "handler(") ||
		strings.Contains(text, "set_callback(") ||
		strings.Contains(text, "set_handler(") ||
		strings.Contains(text, "callback_ =") ||
		strings.Contains(text, "handler_ =")
}

func containsAny(text string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}

	return false
}

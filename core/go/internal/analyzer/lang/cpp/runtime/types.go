package runtime

import "github.com/orurh/patchcourt/internal/model"

const (
	findingRawPointerAsyncCapture    = "cpp.lifetime.raw_pointer_async_capture"
	findingThisCaptureAsync          = "cpp.lifetime.this_capture_async"
	findingThisCaptureStoredCallback = "cpp.lifetime.this_capture_stored_callback"
	findingThisCaptureThread         = "cpp.lifetime.this_capture_thread"
	findingShutdownPolling           = "cpp.shutdown.sleep_polling"
	findingDetachedDelayedCallback   = "cpp.shutdown.detached_delayed_callback"
)

const rawPointerCandidateMaxDistance = 20

type rawPointerCandidate struct {
	name string
	line int
}

type findingBuilder struct {
	finding model.Finding
}

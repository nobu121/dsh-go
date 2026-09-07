package app

import "sync"

const (
	prepStartMsg = productDisplayName
)

// PrepOfferItem is one pending update on the shell prep page.
type PrepOfferItem struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// PrepProgress is emitted to the shell prep page while dsh is being located,
// downloaded, started, or offered an update.
type PrepProgress struct {
	Stage   string          `json:"stage"`
	Message string          `json:"message"`
	Action  string          `json:"action,omitempty"`
	Version string          `json:"version,omitempty"`
	Items   []PrepOfferItem `json:"items,omitempty"`
	Bytes   int64           `json:"bytes"`
	Total   int64           `json:"total"`
}

const (
	prepOfferStage = "offer"

	prepEvent      = "dsh-go:prep"
	prepRetryEvent = "dsh-go:prep-retry"
	prepReadyEvent = "dsh-go:prep-ready"
	prepApplyEvent = "dsh-go:prep-apply"
	prepLaterEvent = "dsh-go:prep-later"
)

type PrepReporter func(PrepProgress)

func reportPrep(fn PrepReporter, p PrepProgress) {
	if fn != nil {
		fn(p)
	}
}

func prepStartProgress() PrepProgress {
	return PrepProgress{Stage: "start", Message: prepStartMsg}
}

type prepState struct {
	mu sync.Mutex
	p  PrepProgress
	ok bool
}

func (s *prepState) store(p PrepProgress) {
	s.mu.Lock()
	s.p = p
	s.ok = true
	s.mu.Unlock()
}

func (s *prepState) last() (PrepProgress, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.p, s.ok
}

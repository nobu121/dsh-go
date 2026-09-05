package main

import "sync"

// PrepProgress is emitted to the shell prep page while dsh is being located,
// downloaded, or started.
type PrepProgress struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
	Bytes   int64  `json:"bytes"`
	Total   int64  `json:"total"`
}

const (
	prepEvent      = "dsh-go:prep"
	prepRetryEvent = "dsh-go:prep-retry"
	prepReadyEvent = "dsh-go:prep-ready"
)

type PrepReporter func(PrepProgress)

func reportPrep(fn PrepReporter, p PrepProgress) {
	if fn != nil {
		fn(p)
	}
}

type prepState struct {
	mu  sync.Mutex
	p   PrepProgress
	ok  bool
	url string
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

func (s *prepState) setURL(url string) {
	s.mu.Lock()
	s.url = url
	s.mu.Unlock()
}

func (s *prepState) readyURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

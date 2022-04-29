package awsvpn

import "sync"

type status struct {
	status int
	mu     sync.RWMutex
}

func (s *status) Update(u int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = u
}

func (s *status) Get() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status
}

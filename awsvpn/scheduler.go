package awsvpn

import "sync"

type scheduler struct {
	wg sync.WaitGroup
}

func (s *scheduler) Run(fn func()) {
	s.wg.Add(1)
	go func() {
		fn()
		s.wg.Done()
	}()
}

func (s *scheduler) Wait() {
	s.wg.Wait()
}

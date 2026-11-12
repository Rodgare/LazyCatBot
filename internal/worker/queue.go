package worker

import (
	"context"
	"sync"
	"time"
)

// freeJobMaxWait bounds how long free jobs can be delayed by a continuous
// stream of premium jobs. Once the oldest free job has waited this long it is
// processed next, so premium traffic can never starve free channels.
const freeJobMaxWait = 45 * time.Second

type queuedJob struct {
	job        KillJob
	premium    bool
	enqueuedAt time.Time
}

// jobScheduler keeps two FIFO queues and always drains premium work first,
// except when a free job has been waiting longer than freeJobMaxWait.
type jobScheduler struct {
	mu      sync.Mutex
	premium []queuedJob
	free    []queuedJob
	notify  chan struct{}
}

func newJobScheduler() *jobScheduler {
	return &jobScheduler{notify: make(chan struct{}, 1)}
}

func (s *jobScheduler) push(job KillJob, premium bool) {
	s.mu.Lock()
	qj := queuedJob{job: job, premium: premium, enqueuedAt: time.Now()}
	if premium {
		s.premium = append(s.premium, qj)
	} else {
		s.free = append(s.free, qj)
	}
	s.mu.Unlock()

	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *jobScheduler) total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.premium) + len(s.free)
}

// pop blocks until a job is available or ctx is cancelled. The bool is false
// only when the context is done.
func (s *jobScheduler) pop(ctx context.Context) (KillJob, bool) {
	for {
		s.mu.Lock()
		if job, ok := s.popLocked(); ok {
			s.mu.Unlock()
			return job, true
		}
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return KillJob{}, false
		case <-s.notify:
		}
	}
}

func (s *jobScheduler) popLocked() (KillJob, bool) {
	if len(s.premium) == 0 && len(s.free) == 0 {
		return KillJob{}, false
	}

	now := time.Now()
	// Hard starvation guard first.
	if len(s.free) > 0 && now.Sub(s.free[0].enqueuedAt) >= freeJobMaxWait {
		job := s.free[0].job
		s.free = s.free[1:]
		return job, true
	}

	if len(s.premium) > 0 {
		job := s.premium[0].job
		s.premium = s.premium[1:]
		return job, true
	}

	job := s.free[0].job
	s.free = s.free[1:]
	return job, true
}

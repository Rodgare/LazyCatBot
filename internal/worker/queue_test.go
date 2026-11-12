package worker

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerPrefersPremium(t *testing.T) {
	s := newJobScheduler()
	s.push(KillJob{KillID: 1}, false)
	s.push(KillJob{KillID: 2}, true)

	ctx := context.Background()
	job, ok := s.pop(ctx)
	if !ok || job.KillID != 2 {
		t.Fatalf("first pop = %+v, want premium job 2", job)
	}
	job, ok = s.pop(ctx)
	if !ok || job.KillID != 1 {
		t.Fatalf("second pop = %+v, want free job 1", job)
	}
}

func TestSchedulerAgingProtectsFreeJobs(t *testing.T) {
	s := newJobScheduler()
	s.push(KillJob{KillID: 1}, false)

	s.mu.Lock()
	s.free[0].enqueuedAt = time.Now().Add(-freeJobMaxWait - time.Second)
	s.mu.Unlock()

	s.push(KillJob{KillID: 2}, true)

	job, ok := s.pop(context.Background())
	if !ok || job.KillID != 1 {
		t.Fatalf("pop = %+v, want aged free job 1", job)
	}
}

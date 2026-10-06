package worker

import "time"

type RateLimiter struct {
	tokens chan struct{}
}

func NewRateLimiter(ratePerSec, burst int) *RateLimiter {
	l := &RateLimiter{tokens: make(chan struct{}, burst)}
	for i := 0; i < burst; i++ {
		l.tokens <- struct{}{}
	}

	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(ratePerSec))
		defer ticker.Stop()
		for range ticker.C {
			select {
			case l.tokens <- struct{}{}:
			default:
			}
		}
	}()

	return l
}

func (l *RateLimiter) Wait() {
	<-l.tokens
}

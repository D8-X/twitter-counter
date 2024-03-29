package twitter

import (
	"sync"
	"time"
)

type ApiRateLimiter interface {
	// Allow attempts to reserve a request and returns true if request can run.
	Allow() bool
	// WaitTime returns duration after which a next request can run.
	WaitTime() time.Duration
	// MarkLimited marks limiter as fully exchausted at the call moment.
	MarkLimited()
	// SetAvailableTime sets the timestamp when the next request can run.
	SetAvailableTime(timestamp int64)
}

func NewRateLimiter(requests int, timeWindow time.Duration) *TwitterRateLimiter {
	return &TwitterRateLimiter{
		requests:   requests,
		timeWindow: timeWindow,
		now:        time.Now,
	}
}

// TwitterRateLimiter is a rate limiter for Twitter API requests. Twitter api
// allows X requests per Y time window and is similaro a sliding window rate
// limiter.
type TwitterRateLimiter struct {
	timeWindow time.Duration
	// How many requests to allow per timeWindow
	requests int

	currentRequestCount int
	mu                  sync.Mutex

	firstRequest time.Time

	now func() time.Time
}

// Allow attempts to reserver a request. It returns true if next request can run
// now.
func (t *TwitterRateLimiter) Allow() bool {
	t.shouldReset()

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.currentRequestCount < t.requests {
		if t.currentRequestCount == 0 {
			t.firstRequest = t.now()
		}

		t.currentRequestCount++
		return true
	}

	return false
}

// WaitTime returns the duration after which a next request can run.
func (t *TwitterRateLimiter) WaitTime() time.Duration {
	if t.currentRequestCount >= t.requests {
		return t.timeWindow - t.now().Sub(t.firstRequest)
	}

	return 0
}

func (t *TwitterRateLimiter) shouldReset() {
	if t.currentRequestCount > 0 && t.now().Sub(t.firstRequest) > t.timeWindow {
		t.mu.Lock()
		t.currentRequestCount = 0
		t.mu.Unlock()
	}
}

// MarkLimited is a special case function which marks current limiter as if it
// would have reached the rate limit. This is useful when we manually want to
// set the limiter to restricted state.
func (t *TwitterRateLimiter) MarkLimited() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.currentRequestCount = t.requests
}

// SetAvailableTime sets timestamp when the next request can run.
func (t *TwitterRateLimiter) SetAvailableTime(timestamp int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.firstRequest = time.Unix(timestamp, 0).Add(-t.timeWindow)
}

package twitter

import (
	"testing"
	"time"
)

func TestTwitterRateLimiter(t *testing.T) {

	t.Run("allows requests", func(t *testing.T) {
		l := NewRateLimiter(10, time.Minute*15)
		for i := 0; i < 10; i++ {
			if !l.Allow() {
				t.Errorf("Expected to allow request %d", i)
			}
		}
	})

	t.Run("does not allow requests", func(t *testing.T) {
		l := NewRateLimiter(10, time.Minute*15)
		for i := 0; i < 10; i++ {
			l.Allow()
		}

		if l.Allow() {
			t.Errorf("Expected to not allow request")
		}
	})

	t.Run("allows requests after time window", func(t *testing.T) {
		l := NewRateLimiter(10, time.Minute*15)
		for i := 0; i < 10; i++ {
			l.Allow()
		}

		if l.Allow() {
			t.Errorf("Expected to not allow request")
		}

		l.now = func() time.Time {
			return time.Now().Add(time.Minute * 16)
		}

		if !l.Allow() {
			t.Errorf("Expected to allow request")
		}
	})

	// Returns correct wait time
	t.Run("returns correct wait time", func(t *testing.T) {
		l := NewRateLimiter(10, time.Minute*15)

		testTime := time.Now()
		// Fixate the time
		l.now = func() time.Time { return testTime }

		for i := 0; i < 10; i++ {
			l.Allow()
		}

		l.now = func() time.Time {
			return testTime.Add(time.Minute)
		}

		if l.WaitTime() != time.Minute*14 {
			t.Errorf("Expected to return correct wait time")
		}

	})

}

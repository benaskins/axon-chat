package chat

import (
	"sync"
	"testing"
	"time"
)

func TestNewPageFetcher_CustomModel(t *testing.T) {
	f := NewPageFetcher(nil, "custom-model:7b")
	if f.model != "custom-model:7b" {
		t.Errorf("got model %q, want %q", f.model, "custom-model:7b")
	}
}

func TestNewPageFetcher_DefaultModel(t *testing.T) {
	f := NewPageFetcher(nil, "")
	if f.model != defaultExtractionModel {
		t.Errorf("got model %q, want default %q", f.model, defaultExtractionModel)
	}
}

func TestPageFetcher_RateLimit_ConcurrentAccess(t *testing.T) {
	// Create a fetcher with no LLM client (we're only testing rate limiting)
	f := NewPageFetcher(nil, "")

	// Set lastFetch to now so the rate limit is active immediately
	f.lastFetch = time.Now()

	const goroutines = 5
	var wg sync.WaitGroup
	starts := make([]time.Time, goroutines)

	// Launch concurrent goroutines that all try to acquire rate limit
	for i := range goroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			f.mu.Lock()
			var wait time.Duration
			if !f.lastFetch.IsZero() {
				elapsed := time.Since(f.lastFetch)
				if elapsed < fetchDelayBetween {
					wait = fetchDelayBetween - elapsed
				}
			}
			f.lastFetch = time.Now().Add(wait)
			f.mu.Unlock()
			if wait > 0 {
				time.Sleep(wait)
			}
			starts[idx] = time.Now()
		}(i)
	}

	wg.Wait()

	// Verify that fetches are spaced at least fetchDelayBetween apart
	// Sort by time
	for i := 0; i < len(starts); i++ {
		for j := i + 1; j < len(starts); j++ {
			if starts[j].Before(starts[i]) {
				starts[i], starts[j] = starts[j], starts[i]
			}
		}
	}

	for i := 1; i < len(starts); i++ {
		gap := starts[i].Sub(starts[i-1])
		// Allow some tolerance for scheduling jitter (500ms of 1s delay)
		if gap < fetchDelayBetween/2 {
			t.Errorf("fetches %d and %d too close together: %v (want >= %v)", i-1, i, gap, fetchDelayBetween/2)
		}
	}
}

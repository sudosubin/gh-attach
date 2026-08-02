package attachments

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"
)

// stubLookup returns a lookup stub + host snapshot; block gates each call.
func stubLookup(block <-chan struct{}) (lookup hostLookupFunc, snapshot func() []string) {
	var mu sync.Mutex
	var called []string

	lookup = func(_ context.Context, host string) ([]string, error) {
		mu.Lock()
		called = append(called, host)
		mu.Unlock()
		if block != nil {
			<-block
		}
		return []string{"127.0.0.1"}, nil
	}
	snapshot = func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(called)
	}
	return lookup, snapshot
}

func TestPrewarmUploadHostDNS_Cloud(t *testing.T) {
	lookup, snapshot := stubLookup(nil)

	prewarmUploadHostDNS(t.Context(), "github.com", lookup)

	if !waitFor(t, func() bool { return len(snapshot()) == len(cloudUploadHosts) }) {
		t.Fatalf("got %v, want lookups for %v", snapshot(), cloudUploadHosts)
	}
	for _, host := range cloudUploadHosts {
		if !slices.Contains(snapshot(), host) {
			t.Errorf("expected lookup for %q, got %v", host, snapshot())
		}
	}
}

func TestPrewarmUploadHostDNS_EnterpriseSkipsLookup(t *testing.T) {
	lookup, snapshot := stubLookup(nil)

	prewarmUploadHostDNS(t.Context(), "github.example.com", lookup)

	// Negative case: no signal to await, so a short grace period stands in.
	time.Sleep(20 * time.Millisecond)
	if got := snapshot(); len(got) != 0 {
		t.Fatalf("enterprise host triggered lookups: %v", got)
	}
}

func TestPrewarmUploadHostDNS_ReturnsWithoutWaiting(t *testing.T) {
	block := make(chan struct{}) // never closed: awaiting a lookup would hang
	defer close(block)
	lookup, _ := stubLookup(block)

	done := make(chan struct{})
	go func() {
		prewarmUploadHostDNS(t.Context(), "github.com", lookup)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("PrewarmUploadHostDNS blocked on the in-flight lookup instead of returning immediately")
	}
}

func waitFor(t *testing.T, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

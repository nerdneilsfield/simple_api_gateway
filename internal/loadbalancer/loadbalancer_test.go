package loadbalancer

import (
	"sync"
	"testing"
	"time"
)

func runWithTimeout(t *testing.T, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock")
	}
}

func TestSingleBackendFailuresNoDeadlock(t *testing.T) {
	lb := NewRoundRobinLoadBalancer([]string{"a"})
	runWithTimeout(t, func() {
		for i := 0; i < 5; i++ {
			lb.ReportFailure("a")
		}
	})
	if got := lb.NextBackend(); got != "a" {
		t.Fatalf("got %q", got)
	}
}

func TestAllUnhealthyStillServes(t *testing.T) {
	lb := NewRoundRobinLoadBalancer([]string{"a", "b"})
	runWithTimeout(t, func() {
		for i := 0; i < 3; i++ {
			lb.ReportFailure("a")
			lb.ReportFailure("b")
		}
	})
	if lb.NextBackend() == "" {
		t.Fatal("no backend after all failed")
	}
}

func TestHealthyBackendSelected(t *testing.T) {
	lb := NewRoundRobinLoadBalancer([]string{"a", "b"})
	for i := 0; i < 3; i++ {
		lb.ReportFailure("a")
	}
	for i := 0; i < 4; i++ {
		if got := lb.NextBackend(); got != "b" {
			t.Fatalf("got %q", got)
		}
	}
}

func TestConcurrentUse(t *testing.T) {
	lb := NewRoundRobinLoadBalancer([]string{"a", "b"})
	runWithTimeout(t, func() {
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 200; j++ {
					b := lb.NextBackend()
					lb.ReportFailure(b)
					lb.ReportSuccess(b, time.Millisecond)
				}
			}()
		}
		wg.Wait()
	})
}

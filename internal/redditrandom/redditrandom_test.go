package redditrandom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIntnUsesLocalSeeds(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `{"data":{"children":[{"data":{"body":"alpha","replies":{"data":{"children":[]}}}},{"data":{"body":"beta"}},{"data":{"body":"alpha"}}]}}`)
	}))
	defer server.Close()

	rr := New(server.Client(), server.URL)
	var values [3]int
	for i := range values {
		var err error
		values[i], err = rr.Intn(context.Background(), 1_000_000_000)
		if err != nil {
			t.Fatalf("Intn at seed %d: %v", i, err)
		}
		if values[i] < 0 || values[i] >= 1_000_000_000 {
			t.Fatalf("Intn returned out-of-range value: %d", values[i])
		}
	}
	if values[0] != values[2] || values[0] == values[1] {
		t.Fatalf("same comment must yield same value, different comment different value: %v", values)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("expected one fetch for three seeds, got %d", got)
	}
	value, err := rr.Intn(context.Background(), 1)
	if err != nil || value != 0 {
		t.Fatalf("Intn(1) = %d, %v; want 0, nil", value, err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("exhausted seeds should trigger another fetch, got %d requests", got)
	}
	for _, bound := range []int{0, -3} {
		if _, err := rr.Intn(context.Background(), bound); err == nil {
			t.Fatalf("Intn(%d) should reject invalid bound", bound)
		}
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("invalid bounds should not fetch seeds: got %d requests", got)
	}
}

func TestIntnReportsSeedErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"status", http.StatusTooManyRequests, "rate limited", "429"},
		{"no comments", http.StatusOK, `{"data":{"children":[]}}`, "no comments"},
		{"malformed JSON", http.StatusOK, "{", "decode Reddit seeds"},
		{"oversized response", http.StatusOK, strings.Repeat("x", maxSeedResponseBytes+1), "size limit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			_, err := New(server.Client(), server.URL).Intn(context.Background(), 10)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Intn error = %v; want containing %q", err, tc.want)
			}
			if tc.name == "malformed JSON" {
				var syntaxErr *json.SyntaxError
				if !errors.As(err, &syntaxErr) {
					t.Fatalf("decode error should preserve JSON cause: %v", err)
				}
			}
		})
	}
}

func TestIntnRecoversAfterFetchFailure(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"data":{"children":[{"data":{"body":"alpha"}}]}}`)
	}))
	defer server.Close()

	rr := New(server.Client(), server.URL)
	if _, err := rr.Intn(context.Background(), 10); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected status error, got %v", err)
	}
	if value, err := rr.Intn(context.Background(), 10); err != nil || value < 0 || value >= 10 {
		t.Fatalf("expected successful retry, got %d, %v", value, err)
	}
}

func TestIntnCancellationDuringFetch(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := New(server.Client(), server.URL).Intn(ctx, 10)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("fetch did not reach local server")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("fetch cancellation error = %v; want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled fetch did not return")
	}
}

func TestIntnCancellationWhileWaitingForSeeds(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		fmt.Fprint(w, `{"data":{"children":[{"data":{"body":"alpha"}}]}}`)
	}))
	defer server.Close()
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })

	rr := New(server.Client(), server.URL)
	first := make(chan error, 1)
	go func() {
		_, err := rr.Intn(context.Background(), 10)
		first <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first fetch did not reach local server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	waiter := make(chan error, 1)
	go func() {
		_, err := rr.Intn(ctx, 10)
		waiter <- err
	}()
	select {
	case err := <-waiter:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waiting cancellation error = %v; want context.DeadlineExceeded", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waiter did not honor its context while fetching was blocked")
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-first:
		if err != nil {
			t.Fatalf("first fetch failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first fetch did not complete")
	}
}

func TestConcurrentIntnSharesOneSeedPool(t *testing.T) {
	const callers = 45
	var mu sync.Mutex
	var active, maximum, requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		requests++
		if active > maximum {
			maximum = active
		}
		mu.Unlock()
		time.Sleep(time.Millisecond)
		fmt.Fprint(w, `{"data":{"children":[{"data":{"body":"alpha"}},{"data":{"body":"beta"}},{"data":{"body":"gamma"}}]}}`)
		mu.Lock()
		active--
		mu.Unlock()
	}))
	defer server.Close()

	rr := New(server.Client(), server.URL)
	start := make(chan struct{})
	results := make(chan error, callers)
	for range callers {
		go func() {
			<-start
			value, err := rr.Intn(context.Background(), 100)
			if err == nil && (value < 0 || value >= 100) {
				err = fmt.Errorf("out-of-range value: %d", value)
			}
			results <- err
		}()
	}
	close(start)
	for range callers {
		if err := <-results; err != nil {
			t.Fatalf("concurrent Intn: %v", err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if requests != callers/3 || maximum != 1 {
		t.Fatalf("expected %d serial fetches, got %d requests and %d simultaneous fetches", callers/3, requests, maximum)
	}
}

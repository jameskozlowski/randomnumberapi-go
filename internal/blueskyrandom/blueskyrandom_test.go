package blueskyrandom

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

const twoPostFeed = `{"feed":[{"post":{"uri":"at://post/a","replyCount":2}},{"post":{"uri":"at://post/b","replyCount":1}}]}`
const onePostFeed = `{"feed":[{"post":{"uri":"at://post/a","replyCount":1}}]}`

func TestIntnUsesRepliesAndRotatesPosts(t *testing.T) {
	var calls []string
	var callsMu sync.Mutex
	recordCall := func(name string) {
		callsMu.Lock()
		calls = append(calls, name)
		callsMu.Unlock()
	}
	getCalls := func() string {
		callsMu.Lock()
		defer callsMu.Unlock()
		return strings.Join(calls, ",")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		switch r.URL.Path {
		case "/xrpc/app.bsky.feed.getFeed":
			if got := r.URL.Query().Get("feed"); got != discoverFeedURI {
				t.Errorf("feed URI = %q", got)
			}
			if got := r.URL.Query().Get("limit"); got != "12" {
				t.Errorf("feed limit = %q", got)
			}
			recordCall("feed")
			fmt.Fprint(w, twoPostFeed)
		case "/xrpc/app.bsky.feed.getPostThread":
			if r.URL.Query().Get("depth") != "1" || r.URL.Query().Get("parentHeight") != "0" {
				t.Errorf("thread query = %s", r.URL.RawQuery)
			}
			switch uri := r.URL.Query().Get("uri"); uri {
			case "at://post/a":
				recordCall("a")
				fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://reply/a1","record":{"text":"alpha"}}},{"post":{"uri":"at://reply/ignored","record":{"text":"  \n "}}},{"post":{"uri":"at://reply/a2","record":{"text":"beta"}}}]}}`)
			case "at://post/b":
				recordCall("b")
				fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://reply/b1","record":{"text":"gamma"}}}]}}`)
			default:
				t.Errorf("unexpected thread URI %q", uri)
			}
		default:
			t.Errorf("unexpected API path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := New(server.Client(), server.URL)
	var values [3]int
	for i := range values {
		value, err := provider.Intn(context.Background(), 1_000_000_000)
		if err != nil || value < 0 || value >= 1_000_000_000 {
			t.Fatalf("Intn call %d = %d, %v", i, value, err)
		}
		values[i] = value
	}
	if values[0] == values[1] {
		t.Errorf("different reply seeds gave the same result: %v", values)
	}
	if got := getCalls(); got != "feed,a,b" {
		t.Errorf("requests = %s, want feed,a,b", got)
	}
	if value, err := provider.Intn(context.Background(), 1); err != nil || value != 0 {
		t.Fatalf("Intn(1) = %d, %v", value, err)
	}
	if got := getCalls(); got != "feed,a,b,feed,a" {
		t.Errorf("exhausted feed requests = %s", got)
	}
	for _, n := range []int{0, -1} {
		if _, err := provider.Intn(context.Background(), n); err == nil {
			t.Errorf("Intn(%d) should reject invalid bound", n)
		}
	}
	if got := getCalls(); got != "feed,a,b,feed,a" {
		t.Errorf("invalid bounds made network requests: %s", got)
	}
}

func TestIntnSkipsUnusableRepliesAndPosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/app.bsky.feed.getFeed":
			fmt.Fprint(w, `{"feed":[{"post":{"uri":"at://skip","replyCount":0}},{"post":{"uri":"at://empty","replyCount":1}},{"post":{"uri":"at://good","replyCount":1}}]}`)
		case "/xrpc/app.bsky.feed.getPostThread":
			switch r.URL.Query().Get("uri") {
			case "at://empty":
				fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://reply/blank","record":{"text":"\t"}}},{"post":{"record":{"text":"no URI"}}}]}}`)
			case "at://good":
				fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://reply/good","record":{"text":"usable"}}}]}}`)
			default:
				t.Errorf("requested a post without replies: %s", r.URL.RawQuery)
			}
		}
	}))
	defer server.Close()
	if value, err := New(server.Client(), server.URL).Intn(context.Background(), 10); err != nil || value < 0 || value >= 10 {
		t.Fatalf("Intn after empty thread = %d, %v", value, err)
	}
}

func TestIntnReportsFeedAndThreadFailures(t *testing.T) {
	cases := []struct {
		name         string
		feedStatus   int
		feedBody     string
		threadStatus int
		threadBody   string
		want         string
	}{
		{name: "feed status", feedStatus: 429, want: "429"},
		{name: "feed malformed", feedBody: "{", want: "decode response"},
		{name: "feed oversized", feedBody: strings.Repeat("x", maxFeedBytes+1), want: "size limit"},
		{name: "feed empty", feedBody: `{"feed":[]}`, want: "no posts with replies"},
		{name: "feed no candidates", feedBody: `{"feed":[{"post":{"uri":"at://post/a","replyCount":0}}]}`, want: "no posts with replies"},
		{name: "thread status", threadStatus: 503, want: "503"},
		{name: "thread malformed", threadBody: "{", want: "decode response"},
		{name: "thread oversized", threadBody: strings.Repeat("x", maxThreadBytes+1), want: "size limit"},
		{name: "thread empty", threadBody: `{"thread":{"replies":[]}}`, want: "no usable replies"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status, body := tc.feedStatus, tc.feedBody
				if r.URL.Path == "/xrpc/app.bsky.feed.getPostThread" {
					status, body = tc.threadStatus, tc.threadBody
				} else if r.URL.Path != "/xrpc/app.bsky.feed.getFeed" {
					t.Errorf("unexpected path %q", r.URL.Path)
				}
				if status != 0 {
					w.WriteHeader(status)
				}
				if body == "" {
					body = onePostFeed
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			_, err := New(server.Client(), server.URL).Intn(context.Background(), 10)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Intn error = %v; want %q", err, tc.want)
			}
			if tc.name == "feed malformed" || tc.name == "thread malformed" {
				var syntaxErr *json.SyntaxError
				if !errors.As(err, &syntaxErr) {
					t.Errorf("syntax error cause lost: %v", err)
				}
			}
		})
	}
}

func TestIntnRetriesAfterStatusFailure(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/xrpc/app.bsky.feed.getFeed" {
			if requests.Add(1) == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			fmt.Fprint(w, onePostFeed)
			return
		}
		fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://reply/a","record":{"text":"hello"}}}]}}`)
	}))
	defer server.Close()
	provider := New(server.Client(), server.URL)
	if _, err := provider.Intn(context.Background(), 10); err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("first call should fail with 429: %v", err)
	}
	if value, err := provider.Intn(context.Background(), 10); err != nil || value < 0 || value >= 10 {
		t.Fatalf("retry = %d, %v", value, err)
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
	result := make(chan error, 1)
	go func() {
		_, err := New(server.Client(), server.URL).Intn(ctx, 10)
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("fetch did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("fetch cancellation = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled fetch did not return")
	}
}

type observedContext struct {
	context.Context
	observed chan struct{}
	once     sync.Once
}

func (ctx *observedContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.observed) })
	return ctx.Context.Done()
}

func TestIntnCancellationWhileWaitingForPool(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/xrpc/app.bsky.feed.getFeed" {
			close(started)
			<-release
			fmt.Fprint(w, onePostFeed)
			return
		}
		fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://reply/first","record":{"text":"first"}}}]}}`)
	}))
	defer server.Close()
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	provider := New(server.Client(), server.URL)
	first := make(chan error, 1)
	go func() {
		_, err := provider.Intn(context.Background(), 10)
		first <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first fetch did not start")
	}
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &observedContext{Context: base, observed: make(chan struct{})}
	second := make(chan error, 1)
	go func() {
		_, err := provider.Intn(ctx, 10)
		second <- err
	}()
	select {
	case <-ctx.observed:
	case <-time.After(5 * time.Second):
		t.Fatal("second call did not enter gate wait")
	}
	cancel()
	select {
	case err := <-second:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiting cancellation = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waiting call did not return")
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case <-first:
	case <-time.After(5 * time.Second):
		t.Fatal("first fetch did not finish")
	}
}

func TestConcurrentIntnSharesPool(t *testing.T) {
	const callers = 36
	var active, maxActive, feeds, threads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if active.Add(1) != 1 {
			maxActive.Store(2)
		}
		defer active.Add(-1)
		time.Sleep(time.Millisecond)
		if r.URL.Path == "/xrpc/app.bsky.feed.getFeed" {
			feeds.Add(1)
			fmt.Fprint(w, twoPostFeed)
			return
		}
		if r.URL.Path != "/xrpc/app.bsky.feed.getPostThread" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		threads.Add(1)
		fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://reply/1","record":{"text":"one"}}},{"post":{"uri":"at://reply/2","record":{"text":"two"}}},{"post":{"uri":"at://reply/3","record":{"text":"three"}}},{"post":{"uri":"at://reply/4","record":{"text":"four"}}}]}}`)
	}))
	defer server.Close()
	provider := New(server.Client(), server.URL)
	start := make(chan struct{})
	result := make(chan error, callers)
	for range callers {
		go func() {
			<-start
			value, err := provider.Intn(context.Background(), 100)
			if err == nil && (value < 0 || value >= 100) {
				err = fmt.Errorf("out-of-range result: %d", value)
			}
			result <- err
		}()
	}
	close(start)
	for range callers {
		if err := <-result; err != nil {
			t.Fatalf("concurrent Intn failed: %v", err)
		}
	}
	if got := feeds.Load(); got != 5 {
		t.Errorf("feed fetches = %d, want 5", got)
	}
	if got := threads.Load(); got != 9 {
		t.Errorf("thread fetches = %d, want 9", got)
	}
	if got := maxActive.Load(); got != 0 {
		t.Errorf("overlapping requests: %d", got)
	}
}

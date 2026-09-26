package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jameskozlowski/randomnumberapi-go/internal/blueskyrandom"
)

func testApp(seedURL string) *api {
	return &api{
		log:         slog.New(slog.NewJSONHandler(io.Discard, nil)),
		blueskyrand: blueskyrandom.New(nil, seedURL),
	}
}

func request(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}

func TestNumberBoundsAndValidation(t *testing.T) {
	handler := testApp("").getRoutes()
	response := request(t, handler, "/api/v1.0/random?min=7&max=8&count=4")
	var numbers []int
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &numbers) != nil || len(numbers) != 4 {
		t.Fatalf("number response: %d %s", response.Code, response.Body.String())
	}
	for _, number := range numbers {
		if number != 7 {
			t.Fatalf("max should be exclusive; got %d", number)
		}
	}
	maxInt := int(^uint(0) >> 1)
	response = request(t, handler, fmt.Sprintf("/api/v1.0/random?min=%d&max=%d", maxInt-1, maxInt))
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &numbers) != nil || len(numbers) != 1 || numbers[0] != maxInt-1 {
		t.Fatalf("large valid bounds: %d %s", response.Code, response.Body.String())
	}
	for _, query := range []string{
		"count=invalid", "count=101", "count=1&count=2", "min=-1", "min=9223372036854775807",
		"min=5&max=5", "min=5&max=4", "secure=maybe",
	} {
		response := request(t, handler, "/api/v1.0/random?"+query)
		var body struct {
			Error string `json:"error"`
		}
		if response.Code != http.StatusBadRequest || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Error == "" {
			t.Errorf("invalid query %q: %d %s", query, response.Code, response.Body.String())
		}
	}
}

func TestSecureNumbersAndStrings(t *testing.T) {
	handler := testApp("").getRoutes()
	response := request(t, handler, "/api/v1.0/random?min=9&max=10&count=100&secure=true")
	var numbers []int
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &numbers) != nil || len(numbers) != 100 {
		t.Fatalf("secure number response: %d %s", response.Code, response.Body.String())
	}
	for _, number := range numbers {
		if number != 9 {
			t.Fatalf("secure number outside [9,10): %d", number)
		}
	}
	response = request(t, handler, "/api/v1.0/randomstring?min=1000&max=1001&count=2&secure=true&all=true")
	var values []string
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &values) != nil || len(values) != 2 {
		t.Fatalf("secure string response: %d %s", response.Code, response.Body.String())
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*"
	for _, value := range values {
		if len(value) != 1000 || strings.Trim(value, alphabet) != "" {
			t.Fatalf("invalid secure string length or character set: length %d", len(value))
		}
	}
	for _, query := range []string{"all=wat", "min=1000", "max=1002", "min=5&max=5", "secure=nope"} {
		response := request(t, handler, "/api/v1.0/randomstring?"+query)
		if response.Code != http.StatusBadRequest {
			t.Errorf("invalid string query %q: %d %s", query, response.Code, response.Body.String())
		}
	}
}

func TestBlueskyNumberEndpoint(t *testing.T) {
	seedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/app.bsky.feed.getFeed":
			fmt.Fprint(w, `{"feed":[{"post":{"uri":"at://did:plc:example/app.bsky.feed.post/abc","replyCount":2}}]}`)
		case "/xrpc/app.bsky.feed.getPostThread":
			fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://did:plc:one/app.bsky.feed.post/one","record":{"text":"first"}}},{"post":{"uri":"at://did:plc:two/app.bsky.feed.post/two","record":{"text":"second"}}}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer seedServer.Close()

	response := request(t, testApp(seedServer.URL).getRoutes(), "/api/v1.0/randomblueskynumber?min=7&max=8&count=2")
	var numbers []int
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &numbers) != nil || len(numbers) != 2 || numbers[0] != 7 || numbers[1] != 7 {
		t.Fatalf("Bluesky-seeded response: %d %s", response.Code, response.Body.String())
	}
}

func TestBlueskyResponseAndPartialFailure(t *testing.T) {
	var feeds, threads int
	seedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/app.bsky.feed.getFeed":
			feeds++
			if feeds > 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			fmt.Fprint(w, `{"feed":[{"post":{"uri":"at://did:plc:example/app.bsky.feed.post/abc","replyCount":1}}]}`)
		case "/xrpc/app.bsky.feed.getPostThread":
			threads++
			fmt.Fprint(w, `{"thread":{"replies":[{"post":{"uri":"at://did:plc:reply/app.bsky.feed.post/xyz","record":{"text":"comment"}}}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer seedServer.Close()

	handler := testApp(seedServer.URL).getRoutes()
	response := request(t, handler, "/api/v1.0/randomblueskynumber?count=2")
	var body struct {
		Error string `json:"error"`
	}
	if response.Code != http.StatusBadGateway || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Error != "Bad Gateway" {
		t.Fatalf("upstream failure must return JSON error only: %d %s", response.Code, response.Body.String())
	}
	if feeds != 2 || threads != 1 {
		t.Fatalf("expected one seed then a failed refill, got %d feed calls and %d thread calls", feeds, threads)
	}
	response = request(t, handler, "/api/v1.0/randomblueskynumber?secure=true")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("Bluesky source cannot satisfy secure=true: %d %s", response.Code, response.Body.String())
	}
	response = request(t, handler, "/api/v1.0/randomredditnumber")
	if response.Code != http.StatusNotFound {
		t.Fatalf("obsolete Reddit route must not report success: %d %s", response.Code, response.Body.String())
	}
}

func TestCrossOriginAPIResponses(t *testing.T) {
	handler := testApp("").getRoutes()
	for _, test := range []struct {
		path string
		want int
	}{
		{"/api/v1.0/random", http.StatusOK},
		{"/api/v1.0/uuid?count=0", http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		req.Header.Set("Origin", "null")
		handler.ServeHTTP(response, req)
		if response.Code != test.want || response.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s: status=%d, CORS=%q", test.path, response.Code, response.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestUUIDHealthAndRequestLogging(t *testing.T) {
	var logs strings.Builder
	app := &api{log: slog.New(slog.NewJSONHandler(&logs, nil))}
	handler := app.getRoutes()
	response := request(t, handler, "/api/v1.0/uuid?count=2&private=do-not-log")
	var values []string
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &values) != nil || len(values) != 2 || values[0] == values[1] {
		t.Fatalf("UUID response: %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Request-ID") == "" || !strings.Contains(logs.String(), `"status":200`) || strings.Contains(logs.String(), "do-not-log") {
		t.Fatalf("missing request ID/status or leaked query: headers=%v logs=%q", response.Header(), logs.String())
	}
	response = request(t, handler, "/healthz")
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("health check: %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1.0/random", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST should be rejected: %d %s", response.Code, response.Body.String())
	}
}

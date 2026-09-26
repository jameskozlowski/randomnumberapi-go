package blueskyrandom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultAPIBaseURL = "https://public.api.bsky.app"

const (
	discoverFeedURI = "at://did:plc:z72i7hdynmk6r22z27h6tvur/app.bsky.feed.generator/whats-hot"
	maxFeedPosts    = 12
	maxFeedBytes    = 1 << 20
	maxThreadBytes  = 2 << 20
)

// BlueskyRandom obtains seeds from public replies to posts in the Discover feed.
// The gate serializes both refills and consumption, but callers can stop waiting
// for it as soon as their context is canceled.
type BlueskyRandom struct {
	client  *http.Client
	baseURL string
	gate    chan struct{}

	seeds    []uint32 // Access only while holding gate.
	nextSeed int
	posts    []string // Feed candidates, accessed only while holding gate.
	nextPost int
}

func New(client *http.Client, baseURL string) *BlueskyRandom {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if baseURL == "" {
		baseURL = DefaultAPIBaseURL
	}
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return &BlueskyRandom{client: client, baseURL: strings.TrimRight(baseURL, "/"), gate: gate}
}

func (br *BlueskyRandom) Intn(ctx context.Context, n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("random upper bound must be positive: %d", n)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-br.gate:
	}
	defer func() { br.gate <- struct{}{} }()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if br.nextSeed == len(br.seeds) {
		if err := br.refill(ctx); err != nil {
			return 0, err
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	seed := br.seeds[br.nextSeed]
	br.nextSeed++
	// A separate PCG per reply makes each result independent of the call order.
	return rand.New(rand.NewPCG(uint64(seed), 0)).IntN(n), nil
}

// refill tries each remaining post at most once. On subsequent refills it
// advances through this feed before fetching another one, rather than always
// selecting its first post.
func (br *BlueskyRandom) refill(ctx context.Context) error {
	if br.nextPost == len(br.posts) {
		posts, err := br.fetchFeed(ctx)
		if err != nil {
			return err
		}
		br.posts = posts
		br.nextPost = 0
	}
	for br.nextPost < len(br.posts) {
		post := br.posts[br.nextPost]
		br.nextPost++
		seeds, err := br.fetchThread(ctx, post)
		if err != nil {
			return err
		}
		if len(seeds) != 0 {
			br.seeds = seeds
			br.nextSeed = 0
			return nil
		}
	}
	return errors.New("Bluesky Discover posts contain no usable replies")
}

func (br *BlueskyRandom) fetchFeed(ctx context.Context) ([]string, error) {
	query := url.Values{"feed": {discoverFeedURI}, "limit": {"12"}}
	var data struct {
		Feed []struct {
			Post struct {
				URI        string `json:"uri"`
				ReplyCount int    `json:"replyCount"`
			} `json:"post"`
		} `json:"feed"`
	}
	if err := br.getJSON(ctx, "/xrpc/app.bsky.feed.getFeed", query, maxFeedBytes, &data); err != nil {
		return nil, fmt.Errorf("fetch Bluesky Discover feed: %w", err)
	}
	posts := make([]string, 0, min(len(data.Feed), maxFeedPosts))
	for _, item := range data.Feed {
		if item.Post.URI != "" && item.Post.ReplyCount > 0 {
			posts = append(posts, item.Post.URI)
			if len(posts) == maxFeedPosts {
				break
			}
		}
	}
	if len(posts) == 0 {
		return nil, errors.New("Bluesky Discover feed contains no posts with replies")
	}
	return posts, nil
}

func (br *BlueskyRandom) fetchThread(ctx context.Context, uri string) ([]uint32, error) {
	query := url.Values{"uri": {uri}, "depth": {"1"}, "parentHeight": {"0"}}
	var data struct {
		Thread struct {
			Replies []struct {
				Post struct {
					URI    string `json:"uri"`
					Record struct {
						Text string `json:"text"`
					} `json:"record"`
				} `json:"post"`
			} `json:"replies"`
		} `json:"thread"`
	}
	if err := br.getJSON(ctx, "/xrpc/app.bsky.feed.getPostThread", query, maxThreadBytes, &data); err != nil {
		return nil, fmt.Errorf("fetch Bluesky thread %s: %w", uri, err)
	}
	seeds := make([]uint32, 0, len(data.Thread.Replies))
	for _, reply := range data.Thread.Replies {
		text := reply.Post.Record.Text
		if reply.Post.URI == "" || strings.TrimSpace(text) == "" {
			continue
		}
		h := fnv.New32a()
		_, _ = io.WriteString(h, reply.Post.URI)
		_, _ = h.Write([]byte{0})
		_, _ = io.WriteString(h, text)
		seeds = append(seeds, h.Sum32())
	}
	return seeds, nil
}

func (br *BlueskyRandom) getJSON(ctx context.Context, path string, query url.Values, maxBytes int64, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, br.baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := br.client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return fmt.Errorf("response exceeds %d-byte size limit", maxBytes)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

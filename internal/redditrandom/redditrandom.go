package redditrandom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

const DefaultSeedURL = "https://api.reddit.com/r/all/comments"
const redditUserAgent = "randomnumberapi/0.1 by JK"
const maxSeedResponseBytes = 1 << 20

type RedditRandom struct {
	client  *http.Client
	seedURL string

	once  sync.Once
	gate  chan struct{}
	seeds []int // Access only while holding gate.
}

func New(client *http.Client, seedURL string) *RedditRandom {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if seedURL == "" {
		seedURL = DefaultSeedURL
	}
	return &RedditRandom{client: client, seedURL: seedURL}
}

func (rr *RedditRandom) Intn(ctx context.Context, n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("random upper bound must be positive: %d", n)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	rr.once.Do(func() {
		rr.gate = make(chan struct{}, 1)
		rr.gate <- struct{}{}
	})
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-rr.gate:
	}
	defer func() { rr.gate <- struct{}{} }()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if len(rr.seeds) == 0 {
		seeds, err := rr.getSeeds(ctx)
		if err != nil {
			return 0, err
		}
		rr.seeds = seeds
	}
	seed := rr.seeds[0]
	rr.seeds = rr.seeds[1:]
	// A new PCG for each comment keeps its result independent of other callers.
	return rand.New(rand.NewPCG(uint64(seed), 0)).IntN(n), nil
}

func (rr *RedditRandom) getSeeds(ctx context.Context) ([]int, error) {
	client := rr.client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	seedURL := rr.seedURL
	if seedURL == "" {
		seedURL = DefaultSeedURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, seedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Reddit seed request: %w", err)
	}
	req.Header.Set("User-Agent", redditUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Reddit seeds: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch Reddit seeds: unexpected status %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSeedResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Reddit seeds: %w", err)
	}
	if len(body) > maxSeedResponseBytes {
		return nil, errors.New("Reddit seed response exceeds size limit")
	}
	var comments redditcomment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("decode Reddit seeds: %w", err)
	}
	if len(comments.Data.Children) == 0 {
		return nil, errors.New("Reddit seed response contains no comments")
	}
	seeds := make([]int, len(comments.Data.Children))
	for i, child := range comments.Data.Children {
		seeds[i] = getHash(child.Data.Body)
	}
	return seeds, nil
}

func getHash(s string) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(s)) // hash.Hash.Write cannot fail.
	return int(hasher.Sum32())
}

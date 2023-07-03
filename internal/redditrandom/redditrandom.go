package redditrandom

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"io"
	"math/rand"
	"net/http"
)

const redditSeedURL = "https://api.reddit.com/r/all/comments"
const redditUserAgent = "randomnumberapi/0.1 by JK"

type RedditRandom struct {
	seeds []int
}

func (rr *RedditRandom) Intn(n int) (int, error) {
	if len(rr.seeds) == 0 {
		seeds, err := rr.getSeeds()
		if err != nil || len(seeds) == 0 {
			return 0, errors.New("error retrieving new seeds")
		}
		rr.seeds = seeds
	}
	seed := rr.seeds[0]
	rr.seeds = rr.seeds[1:]
	r := rand.New(rand.NewSource(int64(seed)))
	return r.Intn(n), nil
}

func (rr *RedditRandom) getSeeds() ([]int, error) {

	var client http.Client
	req, err := http.NewRequest("GET", redditSeedURL, nil)
	if err != nil {
		return nil, errors.New("error connecting to url to retrieve new seeds")
	}
	req.Header.Set("User-Agent", redditUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("error connecting to url to retrieve new seeds")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New("error reading body to retrieve new seeds")
		}
		var comments redditcomment
		err = json.Unmarshal(bodyBytes, &comments)
		if err != nil {
			return nil, errors.New("error linearizing json")
		}
		var seeds []int
		for i := 0; i < len(comments.Data.Children); i++ {
			seeds = append(seeds, getHash(comments.Data.Children[i].Data.Body))
		}
		return seeds, nil
	}
	return nil, errors.New("error getting new seeds, received bad status")
}

var fnvHash = fnv.New32a()

func getHash(s string) int {
	_, err := fnvHash.Write([]byte(s))
	if err != nil {
		return 0
	}
	defer fnvHash.Reset()
	return int(fnvHash.Sum32())
}

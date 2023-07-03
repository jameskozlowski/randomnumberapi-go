package redditrandom

import "testing"

func TestGetSeeds(t *testing.T) {
	rr := RedditRandom{}
	seeds, err := rr.getSeeds()

	if err != nil || len(seeds) != 25 {
		t.Errorf("Error getting seeds")
	}
}

func TestGetRandom(t *testing.T) {
	rr := RedditRandom{}
	r, err := rr.Intn(10)
	if err != nil || r > 10 {
		t.Errorf("Error getting random, expcted < 10 got %d", r)
	}

	r, err = rr.Intn(1)
	if err != nil || r > 1 {
		t.Errorf("Error getting random, expcted < 1 got %d", r)
	}

	r, err = rr.Intn(100)
	if err != nil || r > 100 {
		t.Errorf("Error getting random, expcted < 100 got %d", r)
	}
}

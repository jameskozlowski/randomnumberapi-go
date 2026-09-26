package redditrandom

// redditcomment models only the data used as randomness seeds. Other Reddit
// fields can have different JSON shapes and must not affect seed extraction.
type redditcomment struct {
	Data struct {
		Children []struct {
			Data struct {
				Body string `json:"body"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

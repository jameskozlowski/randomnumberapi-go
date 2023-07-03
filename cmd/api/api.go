package main

import (
	"github.com/jameskozlowski/randomnumberapi-go/internal/logger"
	"github.com/jameskozlowski/randomnumberapi-go/internal/redditrandom"
)

type api struct {
	log        logger.Logger
	redditrand redditrandom.RedditRandom
}

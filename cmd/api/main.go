package main

import (
	"flag"
	"github.com/jameskozlowski/randomnumberapi-go/internal/logger"
	"github.com/jameskozlowski/randomnumberapi-go/internal/redditrandom"
	"net/http"
)

func main() {

	addr := flag.String("addr", ":4000", "HTTP network address")
	log := logger.Logger{}
	log.Init()
	redditrand := redditrandom.RedditRandom{}
	app := &api{
		log:        log,
		redditrand: redditrand,
	}
	mux := app.getRoutes()
	log.LogInfo("Starting server on :4000")
	err := http.ListenAndServe(*addr, mux)
	log.LogFatal(err)
}

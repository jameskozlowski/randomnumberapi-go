package main

import (
	"flag"
	"github.com/jameskozlowski/randomnumberapi-go/internal/logger"
	"net/http"
)

func main() {

	addr := flag.String("addr", ":4000", "HTTP network address")
	log := logger.Logger{}
	log.Init()
	app := &api{
		log: log,
	}
	mux := app.getRoutes()
	log.LogInfo("Starting server on :4000")
	err := http.ListenAndServe(*addr, mux)
	log.LogFatal(err)
}

package main

import "net/http"

func (app *api) getRoutes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("GET /", fileServer)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1.0/random", app.randomNumber)
	mux.HandleFunc("GET /api/v1.0/random/", app.randomNumber)
	mux.HandleFunc("GET /api/v1.0/randomnumber", app.randomNumber)
	mux.HandleFunc("GET /api/v1.0/randomnumber/", app.randomNumber)
	mux.HandleFunc("GET /api/v1.0/uuid", app.randomUUID)
	mux.HandleFunc("GET /api/v1.0/uuid/", app.randomUUID)
	mux.HandleFunc("GET /api/v1.0/randomuuid", app.randomUUID)
	mux.HandleFunc("GET /api/v1.0/randomuuid/", app.randomUUID)
	mux.HandleFunc("GET /api/v1.0/randomstring", app.randomString)
	mux.HandleFunc("GET /api/v1.0/randomstring/", app.randomString)
	mux.HandleFunc("GET /api/v1.0/randomblueskynumber", app.randomBlueskyNumber)
	mux.HandleFunc("GET /api/v1.0/randomblueskynumber/", app.randomBlueskyNumber)
	return app.logRequests(setSecureHeaders(mux))
}

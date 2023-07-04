package main

import "net/http"

func (app *api) getRoutes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/", http.StripPrefix("/", fileServer))
	mux.HandleFunc("/api/v1.0/random", app.randomNumber)
	mux.HandleFunc("/api/v1.0/random/", app.randomNumber)
	mux.HandleFunc("/api/v1.0/randomnumber", app.randomNumber)
	mux.HandleFunc("/api/v1.0/randomnumber/", app.randomNumber)
	mux.HandleFunc("/api/v1.0/uuid", app.randomUUID)
	mux.HandleFunc("/api/v1.0/uuid/", app.randomUUID)
	mux.HandleFunc("/api/v1.0/randomuuid", app.randomUUID)
	mux.HandleFunc("/api/v1.0/randomuuid/", app.randomUUID)
	mux.HandleFunc("/api/v1.0/randomstring", app.randomString)
	mux.HandleFunc("/api/v1.0/randomstring/", app.randomString)
	mux.HandleFunc("/api/v1.0/randomredditnumber", app.randomRedditNumber)
	mux.HandleFunc("/api/v1.0/randomredditnumber/", app.randomRedditNumber)
	return setSecureHeaders(mux)
}

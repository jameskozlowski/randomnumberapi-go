package main

import "net/http"

func (app *api) getRoutes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/api/v1.0/random", app.randomNumber)
	mux.HandleFunc("/api/v1.0/random/", app.randomNumber)
	mux.HandleFunc("/api/v1.0/randomnumber/", app.randomNumber)
	mux.HandleFunc("/api/v1.0/randomnumber", app.randomNumber)
	return setSecureHeaders(mux)
}

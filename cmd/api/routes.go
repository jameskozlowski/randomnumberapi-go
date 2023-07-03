package main

import "net/http"

func getRoutes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/", home)

	return setSecureHeaders(mux)
}

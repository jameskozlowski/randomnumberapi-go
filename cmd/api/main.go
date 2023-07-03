package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {

	addr := flag.String("addr", ":4000", "HTTP network address")

	mux := getRoutes()
	log.Print("Starting server on :4000")
	err := http.ListenAndServe(*addr, mux)
	log.Fatal(err)
}

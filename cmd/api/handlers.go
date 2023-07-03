package main

import (
	"math/rand"
	"net/http"
	"strconv"
)

func (app *api) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Write([]byte("Hello from random API"))
}

func (app *api) randomNumber(w http.ResponseWriter, r *http.Request) {

	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if err != nil || count < 1 || count > 100 {
		count = 1
	}

	min, err := strconv.Atoi(r.URL.Query().Get("min"))
	if err != nil || min < 0 {
		min = 0
	}

	max, err := strconv.Atoi(r.URL.Query().Get("max"))
	if err != nil || max <= min {
		max = min + 100
	}

	var numbers []int
	for i := 0; i < count; i++ {
		numbers = append(numbers, rand.Intn(max-min)+min)
	}
	err = writeJSON(w, numbers)
	if err != nil {
		app.log.LogWarn("Unable to serialize JSON object")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

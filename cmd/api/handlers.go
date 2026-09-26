package main

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (app *api) randomNumber(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	count, err := parseCount(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	min, max, err := parseBounds(q, 0, 100, 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	secure, err := parseBool(q, "secure")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	numbers := make([]int, count)
	for i := range numbers {
		value, err := randomIntn(max-min, secure)
		if err != nil {
			app.serverError(w, r, fmt.Errorf("generate number: %w", err))
			return
		}
		numbers[i] = min + value
	}
	app.respond(w, r, numbers)
}

func (app *api) randomUUID(w http.ResponseWriter, r *http.Request) {
	count, err := parseCount(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	uuids := make([]string, count)
	for i := range uuids {
		value, err := uuid.NewRandom()
		if err != nil {
			app.serverError(w, r, fmt.Errorf("generate UUID: %w", err))
			return
		}
		uuids[i] = value.String()
	}
	app.respond(w, r, uuids)
}

func (app *api) randomString(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	count, err := parseCount(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	min, max, err := parseBounds(q, 10, 10, 1001)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	all, err := parseBool(q, "all")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	secure, err := parseBool(q, "secure")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	strings := make([]string, count)
	for i := range strings {
		length, err := randomIntn(max-min, secure)
		if err != nil {
			app.serverError(w, r, fmt.Errorf("generate string length: %w", err))
			return
		}
		strings[i], err = generateString(length+min, all, secure)
		if err != nil {
			app.serverError(w, r, fmt.Errorf("generate string: %w", err))
			return
		}
	}
	app.respond(w, r, strings)
}

func (app *api) randomBlueskyNumber(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	count, err := parseCount(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	min, max, err := parseBounds(q, 0, 100, 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	secure, err := parseBool(q, "secure")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if secure {
		writeError(w, http.StatusBadRequest, "Bluesky-seeded numbers cannot be secure")
		return
	}
	numbers := make([]int, count)
	for i := range numbers {
		value, err := app.blueskyrand.Intn(r.Context(), max-min)
		if err != nil {
			app.fail(w, r, http.StatusBadGateway, fmt.Errorf("retrieve Bluesky reply seed: %w", err))
			return
		}
		numbers[i] = min + value
	}
	app.respond(w, r, numbers)
}

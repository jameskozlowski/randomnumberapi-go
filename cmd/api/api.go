package main

import (
	"log/slog"
	"net/http"

	"github.com/jameskozlowski/randomnumberapi-go/internal/redditrandom"
)

type api struct {
	log        *slog.Logger
	redditrand *redditrandom.RedditRandom
}

func (app *api) respond(w http.ResponseWriter, r *http.Request, data any) {
	if err := writeJSON(w, http.StatusOK, data); err != nil {
		app.log.Error("write response", "request_id", requestID(r), "error", err)
	}
}

func (app *api) fail(w http.ResponseWriter, r *http.Request, status int, err error) {
	app.log.Error("request failed", "request_id", requestID(r), "status", status, "error", err)
	writeError(w, status, http.StatusText(status))
}

func (app *api) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.fail(w, r, http.StatusInternalServerError, err)
}

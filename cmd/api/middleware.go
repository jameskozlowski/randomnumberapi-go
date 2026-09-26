package main

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

type requestIDKey struct{}

func requestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey{}).(string)
	return id
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (app *api) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		id, err := uuid.NewRandom()
		if err != nil {
			app.log.Error("create request ID", "error", err)
			writeError(recorder, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		w.Header().Set("X-Request-ID", id.String())
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id.String()))
		defer func() {
			if recovered := recover(); recovered != nil {
				app.log.Error("request panic", "request_id", id.String(), "panic", recovered, "stack", string(debug.Stack()))
				if recorder.status == 0 {
					writeError(recorder, http.StatusInternalServerError, "Internal Server Error")
				}
			}
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			route := r.Pattern
			if route == "" {
				route = r.URL.Path
			}
			app.log.Info("http request", "request_id", id.String(), "method", r.Method, "route", route, "status", status, "duration_ms", time.Since(start).Milliseconds())
		}()
		next.ServeHTTP(recorder, r)
	})
}
func setSecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; style-src 'self' cdn.jsdelivr.net; script-src 'self' cdn.jsdelivr.net; img-src 'self' github.blog;")

		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

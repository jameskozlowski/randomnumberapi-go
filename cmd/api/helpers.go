package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	mathrand "math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
)

func writeJSON(w http.ResponseWriter, status int, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(append(body, '\n'))
	return err
}

func writeError(w http.ResponseWriter, status int, message string) {
	_ = writeJSON(w, status, struct {
		Error string `json:"error"`
	}{message})
}

func parseCount(q url.Values) (int, error) {
	return parseInt(q, "count", 1, 1, 100)
}

func parseInt(q url.Values, name string, defaultValue, min, max int) (int, error) {
	values, ok := q[name]
	if !ok {
		return defaultValue, nil
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("%s must occur once", name)
	}
	value, err := strconv.Atoi(values[0])
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("%s must be between %d and %d", name, min, max)
	}
	return value, nil
}

func parseBounds(q url.Values, defaultMin, defaultSpan, maxLimit int) (int, int, error) {
	limit := int(^uint(0) >> 1)
	if maxLimit != 0 {
		limit = maxLimit
	}
	min, err := parseInt(q, "min", defaultMin, 0, limit-1)
	if err != nil {
		return 0, 0, err
	}
	if min > limit-defaultSpan {
		// An omitted max cannot produce a valid default interval.
		if _, provided := q["max"]; !provided {
			return 0, 0, errors.New("min is too large for the default max")
		}
	}
	defaultMax := min
	if min <= limit-defaultSpan {
		defaultMax += defaultSpan
	}
	max, err := parseInt(q, "max", defaultMax, 1, limit)
	if err != nil {
		return 0, 0, err
	}
	if max <= min {
		return 0, 0, errors.New("max must be greater than min (max is exclusive)")
	}
	return min, max, nil
}

func parseBool(q url.Values, name string) (bool, error) {
	values, ok := q[name]
	if !ok {
		return false, nil
	}
	if len(values) != 1 {
		return false, fmt.Errorf("%s must occur once", name)
	}
	value, err := strconv.ParseBool(values[0])
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}
	return value, nil
}

func randomIntn(n int, secure bool) (int, error) {
	if !secure {
		return mathrand.IntN(n), nil
	}
	value, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func generateString(length int, all, secure bool) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const extended = letters + "1234567890!@#$%^&*"
	alphabet := letters
	if all {
		alphabet = extended
	}
	b := make([]byte, length)
	if !secure {
		for i := range b {
			b[i] = alphabet[mathrand.IntN(len(alphabet))]
		}
		return string(b), nil
	}
	// Reject the incomplete tail of the byte range to keep each character uniform.
	cutoff := 256 / len(alphabet) * len(alphabet)
	var batch [256]byte
	for pos := 0; pos < len(b); {
		if _, err := rand.Read(batch[:]); err != nil {
			return "", err
		}
		for _, value := range batch {
			if int(value) < cutoff {
				b[pos] = alphabet[int(value)%len(alphabet)]
				pos++
				if pos == len(b) {
					break
				}
			}
		}
	}
	return string(b), nil
}

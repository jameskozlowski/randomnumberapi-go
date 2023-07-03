package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
)

func writeJSON(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

func generateString(stringLen int, includeNumbersAndSymbols bool) string {

	var letters []rune
	if includeNumbersAndSymbols {
		letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*")
	} else {
		letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	}

	b := make([]rune, stringLen)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

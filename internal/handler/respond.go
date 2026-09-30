package handler

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, errorResponse{Error: msg})
}

// unexported aliases for use within the handler package
func writeJSON(w http.ResponseWriter, status int, v any) { WriteJSON(w, status, v) }
func writeError(w http.ResponseWriter, status int, msg string) { WriteError(w, status, msg) }

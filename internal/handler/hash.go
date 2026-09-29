package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/service"
)

type HashHandler struct {
	svc *service.HashService
}

func NewHashHandler(svc *service.HashService) *HashHandler {
	return &HashHandler{svc: svc}
}

func (h *HashHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /hash", h.hash)
}

type hashRequest struct {
	Password string `json:"password"`
}

type hashResponse struct {
	Hash string `json:"hash"`
}

func (h *HashHandler) hash(w http.ResponseWriter, r *http.Request) {
	var req hashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := h.svc.Hash(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash failed")
		return
	}

	writeJSON(w, http.StatusOK, hashResponse{Hash: hash})
}

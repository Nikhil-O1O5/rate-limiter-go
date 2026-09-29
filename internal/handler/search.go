package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/service"
)

type SearchHandler struct {
	svc *service.SearchService
}

func NewSearchHandler(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /search", h.search)
	mux.HandleFunc("GET /feed", h.feed)
}

type searchRequest struct {
	Query string `json:"query"`
}

func (h *SearchHandler) search(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	users, err := h.svc.Search(r.Context(), req.Query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *SearchHandler) feed(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, err := h.svc.Feed(r.Context(), page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "feed failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

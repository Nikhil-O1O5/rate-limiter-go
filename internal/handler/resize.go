package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/service"
)

const maxUploadSize = 5 << 20 // 5 MB

type ResizeHandler struct {
	svc *service.ResizeService
}

func NewResizeHandler(svc *service.ResizeService) *ResizeHandler {
	return &ResizeHandler{svc: svc}
}

func (h *ResizeHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /resize", h.resize)
}

func (h *ResizeHandler) resize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	width, err := strconv.Atoi(r.FormValue("width"))
	if err != nil || width <= 0 {
		writeError(w, http.StatusBadRequest, "valid width is required")
		return
	}
	height, err := strconv.Atoi(r.FormValue("height"))
	if err != nil || height <= 0 {
		writeError(w, http.StatusBadRequest, "valid height is required")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read image")
		return
	}

	resized, err := h.svc.Resize(data, width, height)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to resize image")
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	w.Write(resized)
}

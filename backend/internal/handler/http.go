// Package handler exposes the HTTP API.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/plantdoc/backend/internal/ai"
	"github.com/plantdoc/backend/internal/service"
)

var allowedImageTypes = []string{"image/jpeg", "image/png", "image/webp", "image/heic", "image/heif"}

type Handler struct {
	svc         *service.DiagnosisService
	maxUploadMB int64
}

func New(svc *service.DiagnosisService, maxUploadMB int64) *Handler {
	return &Handler{svc: svc, maxUploadMB: maxUploadMB}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("POST /api/ask", h.ask)
	mux.HandleFunc("GET /api/consultations", h.recent)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ask accepts multipart/form-data with fields: question (text), image (file, optional).
func (h *Handler) ask(w http.ResponseWriter, r *http.Request) {
	maxBytes := h.maxUploadMB << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1<<20)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "ไฟล์ใหญ่เกินไปหรือรูปแบบข้อมูลไม่ถูกต้อง")
		return
	}

	in := service.AskInput{Question: strings.TrimSpace(r.FormValue("question"))}

	file, header, err := r.FormFile("image")
	switch {
	case err == nil:
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusBadRequest, "อ่านไฟล์รูปไม่สำเร็จ")
			return
		}
		mime := http.DetectContentType(data)
		if ct := header.Header.Get("Content-Type"); strings.HasPrefix(ct, "image/heic") || strings.HasPrefix(ct, "image/heif") {
			mime = ct // DetectContentType doesn't know HEIC
		}
		if !slices.Contains(allowedImageTypes, mime) {
			writeError(w, http.StatusBadRequest, "รองรับเฉพาะไฟล์ JPG, PNG, WEBP, HEIC")
			return
		}
		in.Image = &ai.Image{Data: data, MimeType: mime}
		in.ImageFilename = header.Filename
	case errors.Is(err, http.ErrMissingFile):
		// no image, fine
	default:
		writeError(w, http.StatusBadRequest, "อ่านไฟล์รูปไม่สำเร็จ")
		return
	}

	if in.Question == "" && in.Image == nil {
		writeError(w, http.StatusBadRequest, "กรุณาพิมพ์คำถามหรือแนบรูปภาพ")
		return
	}

	res, err := h.svc.Ask(r.Context(), in)
	if err != nil {
		slog.Error("ask failed", "err", err)
		writeError(w, http.StatusBadGateway, "ระบบวิเคราะห์ขัดข้อง กรุณาลองใหม่อีกครั้ง")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, err := h.svc.Recent(r.Context(), limit)
	if err != nil {
		slog.Error("list consultations failed", "err", err)
		writeError(w, http.StatusInternalServerError, "โหลดข้อมูลไม่สำเร็จ")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

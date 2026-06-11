package main

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
)

const (
	uploadFieldName  = "image"
	maxUploadBytes   = 10 << 20 // 10MB
	appVersion       = "1.0.0"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type uploadResponse struct {
	OK          bool   `json:"ok"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

type errorResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: appVersion,
	})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		getLogger().Warn("upload_parse_failed", "error", err.Error())
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{
			OK:      false,
			Message: "file too large or invalid multipart form",
		})
		return
	}

	file, header, err := r.FormFile(uploadFieldName)
	if err != nil {
		getLogger().Warn("upload_missing_field", "field", uploadFieldName, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, errorResponse{
			OK:      false,
			Message: "missing form field: image",
		})
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		getLogger().Error("upload_read_failed", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			OK:      false,
			Message: "failed to read uploaded file",
		})
		return
	}

	img, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		getLogger().Warn("upload_invalid_image", "filename", header.Filename, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, errorResponse{
			OK:      false,
			Message: "invalid image format, only jpeg and png are supported",
		})
		return
	}

	if format != "jpeg" && format != "png" {
		getLogger().Warn("upload_unsupported_format", "format", format, "filename", header.Filename)
		writeJSON(w, http.StatusBadRequest, errorResponse{
			OK:      false,
			Message: "unsupported image format, only jpeg and png are supported",
		})
		return
	}

	bounds := img.Bounds()
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		if format == "jpeg" {
			contentType = "image/jpeg"
		} else {
			contentType = "image/png"
		}
	}

	getLogger().Info("upload_validated",
		"filename", header.Filename,
		"content_type", contentType,
		"format", format,
		"width", bounds.Dx(),
		"height", bounds.Dy(),
		"size", len(imgBytes),
	)

	writeJSON(w, http.StatusOK, uploadResponse{
		OK:          true,
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        len(imgBytes),
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
	})
}

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
	uploadFieldName    = "image"
	masterFaceField    = "master_face"
	maxUploadBytes     = 10 << 20 // 10MB
	appVersion         = "1.0.0"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type uploadResponse struct {
	OK          bool           `json:"ok"`
	Filename    string         `json:"filename"`
	ContentType string         `json:"contentType"`
	Size        int            `json:"size"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	Detection   DetectionFlags `json:"detection"`
	Codes       []int          `json:"codes"`
}

type initResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
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

func parseImageFromForm(r *http.Request, fieldName string) (image.Image, string, string, int, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return nil, "", "", 0, err
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, "", "", 0, err
	}

	img, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, header.Filename, format, len(imgBytes), err
	}

	if format != "jpeg" && format != "png" {
		return nil, header.Filename, format, len(imgBytes), errUnsupportedFormat
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		if format == "jpeg" {
			contentType = "image/jpeg"
		} else {
			contentType = "image/png"
		}
	}

	return img, header.Filename, contentType, len(imgBytes), nil
}

var errUnsupportedFormat = &formatError{}

type formatError struct{}

func (e *formatError) Error() string { return "unsupported image format" }

func handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		getLogger().Warn("init_parse_failed", "error", err.Error())
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{
			OK:      false,
			Message: "file too large or invalid multipart form",
		})
		return
	}

	img, filename, _, size, err := parseImageFromForm(r, masterFaceField)
	if err != nil {
		getLogger().Warn("init_decode_failed", "filename", filename, "error", err.Error())
		status := http.StatusBadRequest
		message := err.Error()
		if err == errUnsupportedFormat {
			message = "unsupported image format, only jpeg and png are supported"
		} else if filename == "" {
			message = "missing form field: master_face"
		}
		writeJSON(w, status, errorResponse{OK: false, Message: message})
		return
	}

	if err := SetMasterFace(img); err != nil {
		getLogger().Warn("init_master_face_failed", "filename", filename, "size", size, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, errorResponse{
			OK:      false,
			Message: err.Error(),
		})
		return
	}

	getLogger().Info("init_master_face_ok", "filename", filename, "size", size)
	writeJSON(w, http.StatusOK, initResponse{
		OK:      true,
		Message: "master face initialized",
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

	img, filename, contentType, size, err := parseImageFromForm(r, uploadFieldName)
	if err != nil {
		if err == errUnsupportedFormat {
			getLogger().Warn("upload_unsupported_format", "filename", filename)
			writeJSON(w, http.StatusBadRequest, errorResponse{
				OK:      false,
				Message: "unsupported image format, only jpeg and png are supported",
			})
			return
		}
		if filename == "" {
			getLogger().Warn("upload_missing_field", "field", uploadFieldName)
			writeJSON(w, http.StatusBadRequest, errorResponse{
				OK:      false,
				Message: "missing form field: image",
			})
			return
		}
		getLogger().Warn("upload_invalid_image", "filename", filename, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, errorResponse{
			OK:      false,
			Message: "invalid image format, only jpeg and png are supported",
		})
		return
	}

	bounds := img.Bounds()
	detection := AnalyzeImage(img)

	getLogger().Info("upload_validated",
		"filename", filename,
		"content_type", contentType,
		"width", bounds.Dx(),
		"height", bounds.Dy(),
		"size", size,
		"codes", detection.Codes,
	)

	writeJSON(w, http.StatusOK, uploadResponse{
		OK:          true,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Detection:   detection.Flags,
		Codes:       detection.Codes,
	})
}

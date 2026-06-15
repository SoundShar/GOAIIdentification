package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

//go:embed embeddata/yolo11.onnx
var embedYolo11 []byte

//go:embed embeddata/face_detect.onnx
var embedFaceDetect []byte

//go:embed embeddata/face_rec.onnx
var embedFaceRec []byte

var (
	ortLibOnce sync.Once
	ortLibPath string
	ortLibErr  error
)

func resolveOrtLibOverride() string {
	if path := os.Getenv("YKS_ORT_DLL"); path != "" {
		return path
	}
	return os.Getenv("YKS_ORT_LIB")
}

func materializeEmbeddedOrtLib(embedData []byte, cacheFileName string, emptyMsg string) (string, error) {
	if path := resolveOrtLibOverride(); path != "" {
		return path, nil
	}

	ortLibOnce.Do(func() {
		if len(embedData) == 0 {
			ortLibErr = fmt.Errorf("%s", emptyMsg)
			return
		}

		userCache, cacheErr := os.UserCacheDir()
		if cacheErr != nil {
			ortLibErr = fmt.Errorf("resolve user cache dir: %w", cacheErr)
			return
		}
		cacheDir := filepath.Join(userCache, "yks-tool")
		libPath := filepath.Join(cacheDir, cacheFileName)

		if info, statErr := os.Stat(libPath); statErr == nil && info.Size() == int64(len(embedData)) {
			ortLibPath = libPath
			return
		}

		if mkdirErr := os.MkdirAll(cacheDir, 0o755); mkdirErr != nil {
			ortLibErr = fmt.Errorf("create ort cache dir: %w", mkdirErr)
			return
		}

		if writeErr := os.WriteFile(libPath, embedData, 0o644); writeErr != nil {
			ortLibErr = fmt.Errorf("write %s: %w", cacheFileName, writeErr)
			return
		}

		ortLibPath = libPath
	})

	return ortLibPath, ortLibErr
}

func loadEmbeddedModelBytes(name string) ([]byte, error) {
	switch name {
	case "yolo11.onnx":
		if len(embedYolo11) == 0 {
			return nil, fmt.Errorf("embedded yolo11.onnx is empty")
		}
		return embedYolo11, nil
	case "face_detect.onnx":
		if len(embedFaceDetect) == 0 {
			return nil, fmt.Errorf("embedded face_detect.onnx is empty")
		}
		return embedFaceDetect, nil
	case "face_rec.onnx":
		if len(embedFaceRec) == 0 {
			return nil, fmt.Errorf("embedded face_rec.onnx is empty")
		}
		return embedFaceRec, nil
	default:
		return nil, fmt.Errorf("unknown embedded model: %s", name)
	}
}

func loadModelONNX(name string) ([]byte, error) {
	if dir := os.Getenv("YKS_MODEL_DIR"); dir != "" {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read model %s: %w", path, err)
		}
		return data, nil
	}
	return loadEmbeddedModelBytes(name)
}

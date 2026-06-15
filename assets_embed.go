package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// 构建前须执行 scripts/download-deps.ps1，将模型与 onnxruntime.dll 放入 embeddata/

//go:embed embeddata/yolo11.onnx
var embedYolo11 []byte

//go:embed embeddata/face_detect.onnx
var embedFaceDetect []byte

//go:embed embeddata/face_rec.onnx
var embedFaceRec []byte

//go:embed embeddata/onnxruntime.dll
var embedOrtDLL []byte

var (
	ortDLLOnce sync.Once
	ortDLLPath string
	ortDLLErr  error
)

func materializeOrtDLL() (string, error) {
	if path := os.Getenv("YKS_ORT_DLL"); path != "" {
		return path, nil
	}

	ortDLLOnce.Do(func() {
		if len(embedOrtDLL) == 0 {
			ortDLLErr = fmt.Errorf("embedded onnxruntime.dll is empty, run scripts/download-deps.ps1 before build")
			return
		}

		userCache, cacheErr := os.UserCacheDir()
		if cacheErr != nil {
			ortDLLErr = fmt.Errorf("resolve user cache dir: %w", cacheErr)
			return
		}
		cacheDir := filepath.Join(userCache, "yks-tool")
		dllPath := filepath.Join(cacheDir, "onnxruntime.dll")

		if info, statErr := os.Stat(dllPath); statErr == nil && info.Size() == int64(len(embedOrtDLL)) {
			ortDLLPath = dllPath
			return
		}

		if mkdirErr := os.MkdirAll(cacheDir, 0o755); mkdirErr != nil {
			ortDLLErr = fmt.Errorf("create ort cache dir: %w", mkdirErr)
			return
		}

		if writeErr := os.WriteFile(dllPath, embedOrtDLL, 0o644); writeErr != nil {
			ortDLLErr = fmt.Errorf("write onnxruntime.dll: %w", writeErr)
			return
		}

		ortDLLPath = dllPath
	})

	return ortDLLPath, ortDLLErr
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

//go:build windows

package main

import (
	_ "embed"
)

//go:embed embeddata/onnxruntime.dll
var embedOrtLib []byte

func materializeOrtLib() (string, error) {
	return materializeEmbeddedOrtLib(
		embedOrtLib,
		"onnxruntime.dll",
		"embedded onnxruntime.dll is empty, run scripts/download-deps.ps1 before build",
	)
}

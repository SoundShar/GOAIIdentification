//go:build darwin && arm64

package main

import (
	_ "embed"
)

//go:embed embeddata/darwin_arm64/libonnxruntime.dylib
var embedOrtLib []byte

func materializeOrtLib() (string, error) {
	return materializeEmbeddedOrtLib(
		embedOrtLib,
		"libonnxruntime.dylib",
		"embedded arm64 libonnxruntime.dylib is empty, run scripts/download-deps-darwin.sh before build",
	)
}

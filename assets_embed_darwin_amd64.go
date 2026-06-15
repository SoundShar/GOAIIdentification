//go:build darwin && amd64

package main

import (
	_ "embed"
)

//go:embed embeddata/darwin_amd64/libonnxruntime.dylib
var embedOrtLib []byte

func materializeOrtLib() (string, error) {
	return materializeEmbeddedOrtLib(
		embedOrtLib,
		"libonnxruntime.dylib",
		"embedded amd64 libonnxruntime.dylib is empty, run scripts/download-deps-darwin.sh before build",
	)
}

//go:build !cgo

package embed

import (
	"context"
	"errors"
)

// ONNX stands in for the local model in a build without cgo: ONNX Runtime is
// a C library, so it cannot run.
type ONNX struct{}

// NewONNX returns an encoder whose Encode always fails.
func NewONNX(dir string, progress func(ctx context.Context, msg string)) *ONNX { return &ONNX{} }

// Encode says the model cannot run in this build.
func (o *ONNX) Encode(context.Context, string) ([]float64, error) {
	return nil, errors.New("schema-mcp was built without cgo, so the embedding model cannot run")
}

// Close does nothing.
func (o *ONNX) Close() error { return nil }

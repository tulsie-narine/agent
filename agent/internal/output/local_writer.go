package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LocalWriter struct {
	outputPath string
}

func NewLocalWriter(outputPath string) *LocalWriter {
	return &LocalWriter{
		outputPath: outputPath,
	}
}

func (w *LocalWriter) Write(payload interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(w.outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Atomic write: write to temp file first
	tempPath := w.outputPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename temp file to final location
	if err := os.Rename(tempPath, w.outputPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
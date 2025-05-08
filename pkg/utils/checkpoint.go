// pkg/utils/checkpoint.go
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type checkpoint struct {
	LastControlNumber string `json:"lastControlNumber"`
}

// LoadCheckpoint reads checkpoint.json (creates a default file if missing).
func LoadCheckpoint(dir string) (string, error) {
	path := filepath.Join(dir, "checkpoint.json")

	// If it doesn't exist, create with empty control number
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := SaveCheckpoint(dir, ""); err != nil {
			return "", fmt.Errorf("creating default checkpoint: %w", err)
		}
		return "", nil
	} else if err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var cp checkpoint
	if err := json.NewDecoder(f).Decode(&cp); err != nil {
		return "", err
	}
	return cp.LastControlNumber, nil
}

// SaveCheckpoint writes the given control# to checkpoint.json.
func SaveCheckpoint(dir, ctrl string) error {
	path := filepath.Join(dir, "checkpoint.json")
	cp := checkpoint{LastControlNumber: ctrl}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cp)
}

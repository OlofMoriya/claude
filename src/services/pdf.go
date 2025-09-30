package services

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func ReadPDFAsBase64(filePath string) (string, error) {
	// Ensure the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Check if it's a PDF file (basic check by extension)
	if filepath.Ext(filePath) != ".pdf" {
		return "", fmt.Errorf("file is not a PDF: %s", filePath)
	}

	// Read the file
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	// Convert to base64
	base64String := base64.StdEncoding.EncodeToString(fileBytes)

	return base64String, nil
}

package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func saveClipboardImageAsPng() (string, error) {
	// Construct the expected file path with current timestamp
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	fileName := "in_" + time.Now().Format("20060102150405") + ".png"
	filePath := filepath.Join(homeDir, "vision", fileName)
	// Create the directory if it doesn't exist
	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return "", err
	}
	// Construct the osascript command using the file path
	cmdStr := "write (the clipboard as «class PNGf») to (open for access (POSIX file \"" + filePath + "\") with write permission)"

	cmd := exec.Command("osascript", "-e", cmdStr)
	// Execute the command
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return filePath, nil
}

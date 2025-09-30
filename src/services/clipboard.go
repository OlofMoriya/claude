package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.design/x/clipboard"
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

// getImageFromClipboard attempts to read an image from the clipboard
func GetImageFromClipboard() (image.Image, error) {
	// This is platform-specific and might not work on all systems

	// Initialize the clipboard package
	err := clipboard.Init()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize clipboard: %v", err))
	}

	err = clipboard.Init()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize clipboard: %v", err))
		// Handle the error appropriately
	}

	// Read image from clipboard
	imgBytes := clipboard.Read(clipboard.FmtImage)

	if len(imgBytes) == 0 {
		panic("Empty image data")
	}

	// If you need the data as an image.Image type for processing
	img, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		panic(fmt.Sprintf("Failed to decode image: %v", err))
	}
	fmt.Printf("Got image from clipboard in format: %s\n", format)

	return img, nil
}

// imageToBase64 converts an image to a base64-encoded string
func ImageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer

	// Encode image to PNG
	err := png.Encode(&buf, img)
	if err != nil {
		return "", err
	}

	// Encode PNG bytes to base64
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

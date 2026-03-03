package tools

import (
	"fmt"
	"os"
	"os/exec"
	"owl/data"
	"owl/logger"
	"strings"

	"github.com/fatih/color"
)

type NoteTool struct {
}

func (tool *NoteTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (tool *NoteTool) Run(i map[string]string) (string, error) {
	action, ok := i["Action"]
	if !ok || action == "" {
		return "", fmt.Errorf("Action is required")
	}

	// Get the notes directory from environment or use default
	notesDir := os.Getenv("NOTES_DIR")
	if notesDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("Could not determine home directory: %s", err)
		}
		notesDir = fmt.Sprintf("%s/notes", homeDir)
	}

	// Ensure notes directory exists
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return "", fmt.Errorf("Could not create notes directory: %s", err)
	}

	logger.Screen(fmt.Sprintf("Note action: %s", action), color.RGB(150, 150, 150))

	switch strings.ToLower(action) {
	case "search":
		return tool.searchNotes(i, notesDir)
	case "append":
		return tool.appendNote(i, notesDir)
	case "create":
		return tool.createNote(i, notesDir)
	case "read":
		return tool.readNote(i, notesDir)
	default:
		return "", fmt.Errorf("Unknown action '%s'. Valid actions: search, append, create, read", action)
	}
}

func (tool *NoteTool) searchNotes(i map[string]string, notesDir string) (string, error) {
	text, ok := i["Text"]
	if !ok || text == "" {
		return "", fmt.Errorf("Text is required for search action")
	}

	logger.Screen(fmt.Sprintf("\nSearching notes for: %s", text), color.RGB(150, 150, 150))

	commandString := fmt.Sprintln("rg", "--color=never", "--no-heading", "--with-filename", "--line-number", text, notesDir)

	logger.Screen(commandString, color.RGB(150, 150, 150))
	// Use ripgrep to search notes
	cmd := exec.Command("rg", "--color=never", "--no-heading", "--with-filename", "--line-number", text, notesDir)

	out, err := cmd.CombinedOutput()
	logger.Screen(fmt.Sprintf("\nfound:\n%s", out), color.RGB(150, 150, 150))

	if err != nil {
		// rg returns exit code 1 when no matches found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return fmt.Sprintf("No matches found for '%s' in notes directory", text), nil
		}
		return "", fmt.Errorf("Search failed: %s\nOutput: %s", err, string(out))
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return fmt.Sprintf("No matches found for '%s' in notes directory", text), nil
	}

	return fmt.Sprintf("Search results for '%s':\n%s", text, result), nil
}

func (tool *NoteTool) appendNote(i map[string]string, notesDir string) (string, error) {
	name, ok := i["Name"]
	if !ok || name == "" {
		return "", fmt.Errorf("Name is required for append action")
	}

	text, ok := i["Text"]
	if !ok || text == "" {
		return "", fmt.Errorf("Text is required for append action")
	}

	// Security check - ensure name doesn't escape notes directory
	if strings.Contains(name, "..") || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "~") {
		return "", fmt.Errorf("Invalid note name - must be relative to notes directory")
	}

	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	filePath := fmt.Sprintf("%s/%s", notesDir, name)

	logger.Screen(fmt.Sprintf("Appending to note: %s", name), color.RGB(150, 150, 150))

	// Open file in append mode, create if doesn't exist
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("Failed to open note: %s", err)
	}
	defer f.Close()

	// Add newline before text if file is not empty
	fileInfo, _ := f.Stat()
	if fileInfo.Size() > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			return "", fmt.Errorf("Failed to write to note: %s", err)
		}
	}

	if _, err := f.WriteString(text + "\n"); err != nil {
		return "", fmt.Errorf("Failed to write to note: %s", err)
	}

	return fmt.Sprintf("Successfully appended to '%s'", name), nil
}

func (tool *NoteTool) createNote(i map[string]string, notesDir string) (string, error) {
	name, ok := i["Name"]
	if !ok || name == "" {
		return "", fmt.Errorf("Name is required for create action")
	}

	text, ok := i["Text"]
	if !ok || text == "" {
		return "", fmt.Errorf("Text is required for create action")
	}

	// Security check - ensure name doesn't escape notes directory
	if strings.Contains(name, "..") || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "~") {
		return "", fmt.Errorf("Invalid note name - must be relative to notes directory")
	}

	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	filePath := fmt.Sprintf("%s/%s", notesDir, name)

	logger.Screen(fmt.Sprintf("Creating note: %s", name), color.RGB(150, 150, 150))

	// Write file (will overwrite if exists)
	err := os.WriteFile(filePath, []byte(text), 0644)
	if err != nil {
		return "", fmt.Errorf("Failed to create note: %s", err)
	}

	return fmt.Sprintf("Successfully created note '%s' at %s (%d bytes)", name, filePath, len(text)), nil
}

func (tool *NoteTool) readNote(i map[string]string, notesDir string) (string, error) {
	name, ok := i["Name"]
	if !ok || name == "" {
		return "", fmt.Errorf("Name is required for read action")
	}

	// Security check - ensure name doesn't escape notes directory
	if strings.Contains(name, "..") || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "~") {
		return "", fmt.Errorf("Invalid note name - must be relative to notes directory")
	}

	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	filePath := fmt.Sprintf("%s/%s", notesDir, name)

	logger.Screen(fmt.Sprintf("Reading note: %s", name), color.RGB(150, 150, 150))

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("Failed to read note: %s", err)
	}

	return fmt.Sprintf("Content of '%s':\n\n%s", name, string(content)), nil
}

func (tool *NoteTool) GetName() string {
	return "note"
}

func (tool *NoteTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:        tool.GetName(),
		Description: "Interacts with the users note system, create, search, read, append to notes. Notes should contain brief information about domain logic, major arcitechture, notes about important conversations, and logs for important communication such as applications etc. All notes should be taken in markdown format",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Action": {
					Type:        "string",
					Description: "Note action to perform: 'search' (search for a phrase in notes. uses rg), 'append' (adds text to bottom of note), 'create' (writes new file in the notes directory), 'read' (reads note from the directory by filename)",
				},
				"Text": {
					Type:        "string",
					Description: "Text to create, search or append",
				},
				"Name": {
					Type:        "string",
					Description: "Name of file to read, create, or update",
				},
			},
		},
	}, LOCAL
}

func (tool *NoteTool) GetGroups() []string {
	return []string{"dev", "writer"}
}

func init() {
	Register(&NoteTool{})
}

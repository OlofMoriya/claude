package tools

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"owl/data"
	"owl/logger"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

type FileUpdateTool struct {
	RequireApproval bool // Set to true to require user approval before applying changes
}

func (tool *FileUpdateTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (tool *FileUpdateTool) Run(i map[string]string) (string, error) {
	fileName, ok := i["FileName"]
	if !ok {
		return "", fmt.Errorf("FileName parameter is required")
	}

	logger.Screen(fmt.Sprintf("\nAsked to update file %v", fileName), color.RGB(150, 150, 150))

	// Safety checks - same as write_file
	if strings.Contains(fileName, "..") {
		return "", fmt.Errorf("Invalid path '%s' - parent directory references not allowed", fileName)
	}
	if strings.HasPrefix(fileName, "/") {
		return "", fmt.Errorf("Invalid path '%s' - root not allowed", fileName)
	}
	if strings.HasPrefix(fileName, "~") {
		return "", fmt.Errorf("Invalid path '%s' - home not allowed", fileName)
	}

	// Check if file exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return "", fmt.Errorf("File '%s' does not exist. Use write_file to create new files", fileName)
	}

	// Check if approval is required (via env var or tool setting)
	requireApproval := tool.RequireApproval || os.Getenv("OWL_REQUIRE_APPROVAL") == "true"

	// Determine update method based on provided parameters
	if diff, ok := i["Diff"]; ok && diff != "" {
		logger.Screen(fmt.Sprintf("\nAsked to apply diff"), color.RGB(150, 150, 150))

		// Show diff for approval if required
		if requireApproval {
			result, err := tool.requestApproval(fileName, diff, "diff")
			if err != nil {
				return "", fmt.Errorf("Failed to show approval dialog: %s", err)
			}

			switch result {
			case Approved:
				logger.Screen("✓ User approved changes", color.RGB(0, 255, 0))
			case Rejected:
				return "Changes rejected by user", nil
			case Cancelled:
				return "Operation cancelled by user", nil
			}
		}

		return tool.applyDiff(fileName, diff)
	}

	if content, ok := i["Content"]; ok && content != "" {
		// Check for line number method
		if startLine, hasStart := i["StartLine"]; hasStart {
			endLine := i["EndLine"] // optional

			// Generate preview diff for approval
			if requireApproval {
				previewDiff, err := tool.generatePreviewDiff(fileName, content, startLine, endLine, "line")
				if err == nil {
					result, err := tool.requestApproval(fileName, previewDiff, "line-numbers")
					if err != nil {
						return "", fmt.Errorf("Failed to show approval dialog: %s", err)
					}

					switch result {
					case Approved:
						logger.Screen("✓ User approved changes", color.RGB(0, 255, 0))
					case Rejected:
						return "Changes rejected by user", nil
					case Cancelled:
						return "Operation cancelled by user", nil
					}
				}
			}

			return tool.updateByLineNumbers(fileName, content, startLine, endLine)
		}

		// Check for text marker method
		if startText, hasStartText := i["StartText"]; hasStartText {
			endText := i["EndText"] // optional

			// Generate preview diff for approval
			if requireApproval {
				previewDiff, err := tool.generatePreviewDiff(fileName, content, startText, endText, "text")
				if err == nil {
					result, err := tool.requestApproval(fileName, previewDiff, "text-markers")
					if err != nil {
						return "", fmt.Errorf("Failed to show approval dialog: %s", err)
					}

					switch result {
					case Approved:
						logger.Screen("✓ User approved changes", color.RGB(0, 255, 0))
					case Rejected:
						return "Changes rejected by user", nil
					case Cancelled:
						return "Operation cancelled by user", nil
					}
				}
			}

			return tool.updateByTextMarkers(fileName, content, startText, endText)
		}

		return "", fmt.Errorf("Content provided but no update method specified. Use StartLine/EndLine, StartText/EndText, or Diff parameter")
	}

	return "", fmt.Errorf("No update content provided. Use Content with line numbers/text markers, or use Diff parameter")
}

// requestApproval shows the diff to the user and asks for approval
func (tool *FileUpdateTool) requestApproval(fileName, diff, method string) (DiffApprovalResult, error) {
	// Check if we're in a TTY (interactive terminal)
	if !isTerminal() {
		logger.Screen("⚠ Not in interactive terminal, auto-approving", color.RGB(255, 255, 0))
		return Approved, nil
	}

	logger.Screen("\n"+lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFF00")).Render("⚡ Review required - launching diff viewer..."), color.RGB(255, 255, 0))

	// Use the best available viewer
	return ShowDiffWithBestViewer(fileName, diff)
}

// generatePreviewDiff creates a diff preview for line/text methods
func (tool *FileUpdateTool) generatePreviewDiff(fileName, content, start, end, method string) (string, error) {
	// Read the file
	fileContent, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(fileContent), "\n")
	var startLine, endLine int

	if method == "line" {
		startLine, err = strconv.Atoi(start)
		if err != nil {
			return "", err
		}
		if end != "" {
			endLine, err = strconv.Atoi(end)
			if err != nil {
				return "", err
			}
		} else {
			endLine = startLine
		}
	} else if method == "text" {
		// Find text markers
		for i, line := range lines {
			if startLine == 0 && strings.Contains(line, start) {
				startLine = i + 1
			}
			if startLine > 0 && endLine == 0 {
				if end == "" {
					endLine = startLine
					break
				} else if strings.Contains(line, end) {
					endLine = i + 1
					break
				}
			}
		}
	}

	if startLine == 0 || endLine == 0 {
		return "", fmt.Errorf("could not determine lines to update")
	}

	// Build a diff-like preview
	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("--- a/%s\n", fileName))
	diff.WriteString(fmt.Sprintf("+++ b/%s\n", fileName))
	diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", startLine, endLine-startLine+1, startLine, strings.Count(content, "\n")+1))

	// Show context before (up to 3 lines)
	contextStart := max(0, startLine-4)
	for i := contextStart; i < startLine-1 && i < len(lines); i++ {
		diff.WriteString(" " + lines[i] + "\n")
	}

	// Show removed lines
	for i := startLine - 1; i < endLine && i < len(lines); i++ {
		diff.WriteString("-" + lines[i] + "\n")
	}

	// Show added lines
	newLines := strings.Split(content, "\n")
	for _, line := range newLines {
		diff.WriteString("+" + line + "\n")
	}

	// Show context after (up to 3 lines)
	contextEnd := min(len(lines), endLine+3)
	for i := endLine; i < contextEnd; i++ {
		diff.WriteString(" " + lines[i] + "\n")
	}

	return diff.String(), nil
}

// applyDiff applies a unified diff patch to the file
func (tool *FileUpdateTool) applyDiff(fileName, diff string) (string, error) {
	// Create a temporary file for the patch
	tmpFile, err := os.CreateTemp("", "patch-*.diff")
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary patch file: %s", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write the diff to the temporary file
	if _, err := tmpFile.WriteString(diff); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("Failed to write patch content: %s", err)
	}
	tmpFile.Close()

	// Try to apply the patch using git apply
	cmd := exec.Command("git", "apply", "--unidiff-zero", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If git apply fails, try with patch command
		cmd = exec.Command("patch", "-p0", fileName)
		patchInput := strings.NewReader(diff)
		cmd.Stdin = patchInput
		output, err = cmd.CombinedOutput()

		if err != nil {
			return "", fmt.Errorf("Failed to apply patch: %s\nOutput: %s", err, string(output))
		}
	}

	result := fmt.Sprintf("Successfully applied diff to '%s'\n%s", fileName, string(output))
	logger.Screen(result, color.RGB(150, 150, 150))
	return result, nil
}

// updateByLineNumbers updates specific lines in the file
func (tool *FileUpdateTool) updateByLineNumbers(fileName, content, startLineStr, endLineStr string) (string, error) {
	startLine, err := strconv.Atoi(startLineStr)
	if err != nil {
		return "", fmt.Errorf("Invalid StartLine: %s", err)
	}

	// Read the file
	fileContent, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("Failed to read file: %s", err)
	}

	lines := strings.Split(string(fileContent), "\n")

	if startLine < 1 || startLine > len(lines) {
		return "", fmt.Errorf("StartLine %d is out of range (file has %d lines)", startLine, len(lines))
	}

	var endLine int
	if endLineStr != "" {
		endLine, err = strconv.Atoi(endLineStr)
		if err != nil {
			return "", fmt.Errorf("Invalid EndLine: %s", err)
		}
		if endLine < startLine || endLine > len(lines) {
			return "", fmt.Errorf("EndLine %d is invalid (must be between %d and %d)", endLine, startLine, len(lines))
		}
	} else {
		endLine = startLine
	}

	// Split new content into lines
	newLines := strings.Split(content, "\n")

	// Build the new file content
	var result []string
	result = append(result, lines[:startLine-1]...) // Lines before update
	result = append(result, newLines...)            // New content
	result = append(result, lines[endLine:]...)     // Lines after update

	// Write back to file
	newContent := strings.Join(result, "\n")
	if err := os.WriteFile(fileName, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("Failed to write file: %s", err)
	}

	linesChanged := endLine - startLine + 1
	msg := fmt.Sprintf("Successfully updated '%s': replaced %d line(s) (lines %d-%d) with %d line(s)",
		fileName, linesChanged, startLine, endLine, len(newLines))
	logger.Screen(msg, color.RGB(150, 150, 150))
	return msg, nil
}

// updateByTextMarkers updates content between text markers
func (tool *FileUpdateTool) updateByTextMarkers(fileName, content, startText, endText string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", fmt.Errorf("Failed to open file: %s", err)
	}
	defer file.Close()

	var lines []string
	var startLine, endLine int
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Read file and find markers
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		if startLine == 0 && strings.Contains(line, startText) {
			startLine = lineNum
		}
		if startLine > 0 && endLine == 0 {
			if endText == "" {
				// If no end text, update just the start line
				endLine = lineNum
			} else if strings.Contains(line, endText) {
				endLine = lineNum
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("Error reading file: %s", err)
	}

	if startLine == 0 {
		return "", fmt.Errorf("StartText '%s' not found in file", startText)
	}

	if endText != "" && endLine == 0 {
		return "", fmt.Errorf("EndText '%s' not found in file after StartText", endText)
	}

	if endLine == 0 {
		endLine = startLine
	}

	// Split new content into lines
	newLines := strings.Split(content, "\n")

	// Build the new file content
	var result []string
	result = append(result, lines[:startLine-1]...) // Lines before update
	result = append(result, newLines...)            // New content
	result = append(result, lines[endLine:]...)     // Lines after update

	// Write back to file
	newContent := strings.Join(result, "\n")
	if err := os.WriteFile(fileName, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("Failed to write file: %s", err)
	}

	linesChanged := endLine - startLine + 1
	msg := fmt.Sprintf("Successfully updated '%s': replaced %d line(s) (lines %d-%d) with %d line(s)",
		fileName, linesChanged, startLine, endLine, len(newLines))
	logger.Screen(msg, color.RGB(150, 150, 150))
	return msg, nil
}

func (tool *FileUpdateTool) GetName() string {
	return "update_file"
}

func (tool *FileUpdateTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
		Description: "Updates specific parts of an existing file. Supports three methods: 1) Git-style unified diff (recommended for AI models), 2) Line number ranges, 3) Text markers. Cannot create new files - use write_file for that. Path is relative to current working directory. Parent directory references (..) are not allowed for security.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"FileName": {
					Type:        "string",
					Description: "The name/path of the file to update (relative to current directory). Never access any parent directory, root, or home. File must already exist.",
				},
				"Diff": {
					Type:        "string",
					Description: "A unified diff format patch to apply. This is the RECOMMENDED method for AI models as it's precise and shows context. Example format:\n--- a/file.txt\n+++ b/file.txt\n@@ -1,3 +1,3 @@\n line1\n-old line\n+new line\n line3\nIf provided, other parameters are ignored.",
				},
				"Content": {
					Type:        "string",
					Description: "The new content to insert/replace. Required if not using Diff. Must be combined with either StartLine/EndLine or StartText/EndText.",
				},
				"StartLine": {
					Type:        "string",
					Description: "Line number where update should start (1-indexed). Use with Content parameter. If EndLine not provided, replaces only this line.",
				},
				"EndLine": {
					Type:        "string",
					Description: "Line number where update should end (1-indexed, inclusive). Use with StartLine and Content. Lines from StartLine to EndLine will be replaced.",
				},
				"StartText": {
					Type:        "string",
					Description: "Text marker to find where update should start. The line containing this text will be replaced. Use with Content parameter.",
				},
				"EndText": {
					Type:        "string",
					Description: "Text marker to find where update should end (inclusive). Use with StartText and Content. If not provided, only the StartText line is replaced.",
				},
			},
		},
	}
}

// Helper functions
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	Register(&FileUpdateTool{})
}

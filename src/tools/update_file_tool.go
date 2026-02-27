package tools

import (
	"fmt"
	"os"
	"os/exec"
	"owl/data"
	"owl/logger"
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

	diff, ok := i["Diff"]
	if !ok || diff == "" {
		return "", fmt.Errorf("Diff parameter is required")
	}

	logger.Screen(fmt.Sprintf("\nAsked to apply diff"), color.RGB(150, 150, 150))

	// Show diff for approval if required
	if requireApproval {
		result, err := tool.requestApproval(fileName, diff)
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
	result, err := tool.applyDiff(fileName, diff)
	if err != nil {
		logger.Screen(fmt.Sprintf("Failed applying the diff: %s", err), color.RGB(250, 150, 150))
		return "Operation was unsuccessful", err
	}

	return result, nil
}

// requestApproval shows the diff to the user and asks for approval
func (tool *FileUpdateTool) requestApproval(fileName, diff string) (DiffApprovalResult, error) {
	// Check if we're in a TTY (interactive terminal)
	if !isTerminal() {
		logger.Screen("⚠ Not in interactive terminal, auto-approving", color.RGB(255, 255, 0))
		return Approved, nil
	}

	logger.Screen("\n"+lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFF00")).Render("⚡ Review required - launching diff viewer..."), color.RGB(255, 255, 0))

	// Use the best available viewer
	return ShowDiffWithBestViewer(fileName, diff)
}

// applyDiff applies a unified diff patch to the file
func (tool *FileUpdateTool) applyDiff(fileName, diff string) (string, error) {
	// Create a temporary file for the patch
	tmpFile, err := os.CreateTemp("", "patch-*.diff")
	if err != nil {
		logger.Screen(fmt.Sprintf("\nfailed to write temporary file"), color.RGB(150, 150, 150))
		return "", fmt.Errorf("Failed to create temporary patch file: %s", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write the diff to the temporary file
	if _, err := tmpFile.WriteString(diff); err != nil {
		tmpFile.Close()
		logger.Screen(fmt.Sprintf("\nfailed to write patch"), color.RGB(150, 150, 150))
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
		logger.Screen(fmt.Sprintf("\nfailed to apply diff, testing patch"), color.RGB(150, 150, 150))

		if err != nil {
			logger.Screen(fmt.Sprintf("\nfailed to apply patch"), color.RGB(150, 150, 150))
			return "", fmt.Errorf("Failed to apply patch: %s\nOutput: %s", err, string(output))
		}
	}

	result := fmt.Sprintf("Successfully applied diff to '%s'\n%s", fileName, string(output))
	logger.Screen(result, color.RGB(150, 150, 150))
	return result, nil
}

func (tool *FileUpdateTool) GetName() string {
	return "update_file"
}

func (tool *FileUpdateTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:        tool.GetName(),
		Description: "Updates specific parts of an existing file using a Git-style unified diff. This is the RECOMMENDED method for AI models as it's precise and shows context. Cannot create new files - use write_file for that. Path is relative to current working directory. Parent directory references (..) are not allowed for security.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"FileName": {
					Type:        "string",
					Description: "The name/path of the file to update (relative to current directory). Never access any parent directory, root, or home. File must already exist.",
				},
				"Diff": {
					Type:        "string",
					Description: "A unified diff format patch to apply. Example format:\n--- a/file.txt\n+++ b/file.txt\n@@ -1,3 +1,3 @@\n line1\n-old line\n+new line\n line3\n",
				},
			},
		},
	}, LOCAL
}

func (tool *FileUpdateTool) GetGroups() []string {
	return []string{"dev", "writer"}
}

// Helper functions
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func init() {
	Register(&FileUpdateTool{})
}

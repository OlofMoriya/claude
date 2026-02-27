package tools

import (
	"fmt"
	"os/exec"
	"owl/data"
	"owl/logger"
	"strings"

	"github.com/fatih/color"
)

type GitStatusTool struct {
}

func (tool *GitStatusTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (tool *GitStatusTool) Run(i map[string]string) (string, error) {
	action := i["Action"]
	if action == "" {
		action = "status"
	}

	logger.Screen(fmt.Sprintf("Asked to use git with action %v", action), color.RGB(150, 150, 150))

	var cmd *exec.Cmd

	switch strings.ToLower(action) {
	case "status":
		cmd = exec.Command("git", "status", "--short")
	case "branch":
		cmd = exec.Command("git", "branch", "--show-current")
	case "log":
		limit := i["Limit"]
		if limit == "" {
			limit = "10"
		}
		cmd = exec.Command("git", "log", "--oneline", "-n", limit)
	case "diff":
		cmd = exec.Command("git", "diff", "--stat")
	case "uncommitted":
		cmd = exec.Command("git", "diff", "HEAD")
	default:
		return "", fmt.Errorf("Unknown action: %s. Valid actions: status, branch, log, diff, uncommitted", action)
	}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Git command failed: %s", err)
	}

	result := string(out)
	if result == "" {
		return fmt.Sprintf("No output for git %s (this might be normal)", action), nil
	}

	return result, nil
}

func (tool *GitStatusTool) GetName() string {
	return "git_info"
}

func (tool *GitStatusTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:        tool.GetName(),
		Description: "Executes git commands to get repository information. Can show status, current branch, recent commits, and diffs. Useful for understanding the state of the codebase and recent changes.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Action": {
					Type:        "string",
					Description: "Git action to perform: 'status' (show changed files), 'branch' (current branch), 'log' (recent commits), 'diff' (changes summary), 'uncommitted' (full diff of changes). Defaults to 'status'.",
				},
				"Limit": {
					Type:        "string",
					Description: "For 'log' action, number of commits to show. Defaults to '10'.",
				},
			},
		},
	}, LOCAL
}

func (tool *GitStatusTool) GetGroups() []string {
	return []string{"dev"}
}

func init() {
	Register(&GitStatusTool{})
}

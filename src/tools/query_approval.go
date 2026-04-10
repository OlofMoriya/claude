package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ShowQueryApproval shows query text and asks for yes/no approval.
// It uses a tmux popup when available, with terminal fallback.
func ShowQueryApproval(title, content string) (DiffApprovalResult, error) {
	if strings.TrimSpace(os.Getenv("TMUX")) != "" {
		result, err := showQueryApprovalInTmux(title, content)
		if err == nil {
			return result, nil
		}
	}

	return showQueryApprovalInTerminal(title, content)
}

func showQueryApprovalInTmux(title, content string) (DiffApprovalResult, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return Cancelled, fmt.Errorf("tmux is not available: %w", err)
	}

	tmpDir := os.TempDir()
	contentFile := filepath.Join(tmpDir, "owl-dbask-query-preview.txt")
	responseFile := filepath.Join(tmpDir, "owl-dbask-query-response.txt")
	scriptFile := filepath.Join(tmpDir, "owl-dbask-query-approval.sh")

	os.Remove(responseFile)

	if err := os.WriteFile(contentFile, []byte(content), 0644); err != nil {
		return Cancelled, fmt.Errorf("failed to write query preview file: %w", err)
	}
	defer os.Remove(contentFile)

	script := fmt.Sprintf(`#!/bin/bash
set -u

clear
echo "============================================================"
echo " DBASK QUERY APPROVAL: %s"
echo "============================================================"
echo ""
cat "%s"
echo ""
echo "------------------------------------------------------------"
echo "Approve this query?"
echo "  [y] yes"
echo "  [n] no"

while true; do
  echo -n "Your choice: "
  read -n 1 -r REPLY
  echo ""
  case "$REPLY" in
    y|Y)
      echo "approved" > "%s"
      break
      ;;
    n|N)
      echo "rejected" > "%s"
      break
      ;;
    *)
      echo "Please type y or n."
      ;;
  esac
done
`, title, contentFile, responseFile, responseFile)

	if err := os.WriteFile(scriptFile, []byte(script), 0755); err != nil {
		return Cancelled, fmt.Errorf("failed to write approval script: %w", err)
	}
	defer os.Remove(scriptFile)

	cmd := exec.Command(
		"tmux", "display-popup",
		"-E",
		"-w", "90%",
		"-h", "90%",
		"-T", " DBASK Query Approval ",
		scriptFile,
	)

	if err := cmd.Run(); err != nil {
		return Cancelled, fmt.Errorf("failed to launch tmux popup: %w", err)
	}

	responseBytes, err := os.ReadFile(responseFile)
	if err != nil {
		return Cancelled, nil
	}
	defer os.Remove(responseFile)

	switch strings.TrimSpace(string(responseBytes)) {
	case "approved":
		return Approved, nil
	case "rejected":
		return Rejected, nil
	default:
		return Cancelled, nil
	}
}

func showQueryApprovalInTerminal(title, content string) (DiffApprovalResult, error) {
	fmt.Println("============================================================")
	fmt.Printf(" DBASK QUERY APPROVAL: %s\n", title)
	fmt.Println("============================================================")
	fmt.Println()
	fmt.Println(content)
	fmt.Println()
	fmt.Println("------------------------------------------------------------")
	fmt.Println("Approve this query?")
	fmt.Println("  [y] yes")
	fmt.Println("  [n] no")

	for {
		fmt.Print("Your choice: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return Cancelled, err
		}
		switch strings.ToLower(strings.TrimSpace(response)) {
		case "y", "yes":
			return Approved, nil
		case "n", "no":
			return Rejected, nil
		default:
			fmt.Println("Please type y or n.")
		}
	}
}

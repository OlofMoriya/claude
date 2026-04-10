package tools

import (
	"strings"
	"testing"
)

func TestValidateUnifiedDiff_Valid(t *testing.T) {
	diff := strings.Join([]string{
		"--- a/file.txt",
		"+++ b/file.txt",
		"@@ -1,1 +1,1 @@",
		"-old line",
		"+new line",
		"",
	}, "\n")

	if err := validateUnifiedDiff(diff); err != nil {
		t.Fatalf("expected valid diff, got error: %v", err)
	}
}

func TestValidateUnifiedDiff_MissingPrefixInHunkBody(t *testing.T) {
	diff := strings.Join([]string{
		"--- a/file.txt",
		"+++ b/file.txt",
		"@@ -1,1 +1,1 @@",
		"func brokenLine() {}",
		"",
	}, "\n")

	err := validateUnifiedDiff(diff)
	if err == nil {
		t.Fatalf("expected validation error for malformed hunk body")
	}

	if !strings.Contains(err.Error(), "malformed hunk body") {
		t.Fatalf("expected malformed hunk body error, got: %v", err)
	}
}

func TestValidateUnifiedDiff_HunkCountMismatch(t *testing.T) {
	diff := strings.Join([]string{
		"--- a/file.txt",
		"+++ b/file.txt",
		"@@ -1,1 +1,2 @@",
		"-old line",
		"+new line",
		"",
	}, "\n")

	err := validateUnifiedDiff(diff)
	if err == nil {
		t.Fatalf("expected validation error for hunk count mismatch")
	}

	if !strings.Contains(err.Error(), "hunk count mismatch") {
		t.Fatalf("expected hunk count mismatch error, got: %v", err)
	}
}

func TestValidateUnifiedDiff_EmptyFirstHunkThenAnotherHunk(t *testing.T) {
	diff := strings.Join([]string{
		"--- a/file.txt",
		"+++ b/file.txt",
		"@@ -5,6 +5,7 @@ import (",
		"@@ -10,1 +10,1 @@",
		"-old line",
		"+new line",
		"",
	}, "\n")

	err := validateUnifiedDiff(diff)
	if err == nil {
		t.Fatalf("expected validation error for empty first hunk")
	}

	if !strings.Contains(err.Error(), "hunk count mismatch") {
		t.Fatalf("expected hunk count mismatch error, got: %v", err)
	}
}

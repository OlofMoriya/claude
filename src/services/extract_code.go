package services

import (
	"regexp"
	"strings"
)

func ExtractCodeBlocks(markdown string) []string {
	// Regex to match code blocks (```...```)
	// This handles both with and without language specifiers
	re := regexp.MustCompile("(?s)```(?:\\w*\\n|\\n)(.*?)```")

	matches := re.FindAllStringSubmatch(markdown, -1)

	var codeBlocks []string
	for _, match := range matches {
		if len(match) >= 2 {
			// Trim any leading/trailing whitespace
			code := strings.TrimSpace(match[1])
			codeBlocks = append(codeBlocks, code)
		}
	}

	return codeBlocks
}

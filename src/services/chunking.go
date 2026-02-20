package services

import (
	"strings"
	"unicode/utf8"
)

const (
	// Optimal chunk size for embeddings (in characters)
	// ~1000 tokens = ~4000 characters for English text
	OptimalChunkSize = 4000
	
	// Maximum chunk size before forced split
	// ~2000 tokens = ~8000 characters
	MaxChunkSize = 8000
	
	// Minimum chunk size to avoid too small fragments
	MinChunkSize = 500
)

// ChunkMarkdown splits markdown content into semantically meaningful chunks
// optimized for RAG with 1536-dimension embeddings
func ChunkMarkdown(content string) []string {
	if content == "" {
		return []string{}
	}

	// First pass: split on markdown headers
	initialChunks := splitOnHeaders(content)
	
	// Second pass: check sizes and split large chunks
	var finalChunks []string
	for _, chunk := range initialChunks {
		if utf8.RuneCountInString(chunk) <= MaxChunkSize {
			// Chunk is acceptable size
			finalChunks = append(finalChunks, chunk)
		} else {
			// Chunk is too large, need to split further
			subChunks := splitLargeChunk(chunk)
			finalChunks = append(finalChunks, subChunks...)
		}
	}
	
	return finalChunks
}

// splitOnHeaders splits content on markdown headers (##, ###, etc.)
func splitOnHeaders(content string) []string {
	lines := strings.Split(content, "\n")
	var chunks []string
	var currentChunk strings.Builder
	
	for _, line := range lines {
		// Check if line starts with markdown header (## or ###)
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			// Save previous chunk if it has content
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}
			// Start new chunk with the header
			currentChunk.WriteString(line)
			currentChunk.WriteString("\n")
		} else {
			currentChunk.WriteString(line)
			currentChunk.WriteString("\n")
		}
	}
	
	// Add the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	return chunks
}

// splitLargeChunk splits a large chunk into smaller pieces
// Tries to split on paragraph boundaries first, then on sentences
func splitLargeChunk(chunk string) []string {
	var result []string
	
	// Try splitting on double newlines (paragraphs)
	paragraphs := strings.Split(chunk, "\n\n")
	
	var currentChunk strings.Builder
	var header string
	
	// Extract header if present
	lines := strings.Split(chunk, "\n")
	if len(lines) > 0 && (strings.HasPrefix(lines[0], "## ") || strings.HasPrefix(lines[0], "### ")) {
		header = lines[0] + "\n\n"
	}
	
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		
		// Check if adding this paragraph would exceed optimal size
		testChunk := currentChunk.String() + "\n\n" + para
		if utf8.RuneCountInString(testChunk) > OptimalChunkSize && currentChunk.Len() > MinChunkSize {
			// Save current chunk and start new one
			result = append(result, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
			
			// Add header to new chunk for context
			if header != "" {
				currentChunk.WriteString(header)
			}
			currentChunk.WriteString(para)
		} else {
			// Add paragraph to current chunk
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
		}
		
		// Hard limit: if current chunk exceeds max size, force split
		if utf8.RuneCountInString(currentChunk.String()) > MaxChunkSize {
			// Split on sentences
			sentences := splitOnSentences(currentChunk.String())
			var sentenceChunk strings.Builder
			
			for _, sentence := range sentences {
				if utf8.RuneCountInString(sentenceChunk.String()+sentence) > MaxChunkSize && sentenceChunk.Len() > 0 {
					result = append(result, strings.TrimSpace(sentenceChunk.String()))
					sentenceChunk.Reset()
					if header != "" {
						sentenceChunk.WriteString(header)
					}
				}
				sentenceChunk.WriteString(sentence)
			}
			
			currentChunk.Reset()
			currentChunk.WriteString(sentenceChunk.String())
		}
	}
	
	// Add remaining content
	if currentChunk.Len() > 0 {
		result = append(result, strings.TrimSpace(currentChunk.String()))
	}
	
	return result
}

// splitOnSentences splits text on sentence boundaries
func splitOnSentences(text string) []string {
	var sentences []string
	var current strings.Builder
	
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])
		
		// Check for sentence endings
		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			// Look ahead to see if followed by space and capital letter
			if i+2 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '\n') {
				if i+2 < len(runes) && isCapital(runes[i+2]) {
					sentences = append(sentences, current.String())
					current.Reset()
				}
			} else if i+1 == len(runes) {
				// End of text
				sentences = append(sentences, current.String())
				current.Reset()
			}
		}
	}
	
	// Add any remaining text
	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}
	
	return sentences
}

// isCapital checks if a rune is an uppercase letter
func isCapital(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// ChunkText provides a simpler chunking for non-markdown text
func ChunkText(content string, maxSize int) []string {
	if maxSize <= 0 {
		maxSize = OptimalChunkSize
	}
	
	var chunks []string
	var current strings.Builder
	
	paragraphs := strings.Split(content, "\n\n")
	
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		
		if utf8.RuneCountInString(current.String()+"\n\n"+para) > maxSize && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}
	
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	
	return chunks
}

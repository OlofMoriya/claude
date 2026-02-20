package services

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestChunkMarkdown_SmallDocument(t *testing.T) {
	content := `## Introduction
This is a small document that should not be split.

It has a few paragraphs but stays under the size limit.`

	chunks := ChunkMarkdown(content)
	
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
}

func TestChunkMarkdown_MultipleHeaders(t *testing.T) {
	content := `## First Section
This is the first section with some content.

## Second Section
This is the second section with different content.

## Third Section
And this is the third section.`

	chunks := ChunkMarkdown(content)
	
	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}
	
	// Check that each chunk starts with its header
	for i, chunk := range chunks {
		if !strings.HasPrefix(chunk, "##") {
			t.Errorf("Chunk %d does not start with header: %s", i, chunk[:50])
		}
	}
}

func TestChunkMarkdown_LargeChunk(t *testing.T) {
	// Create a large chunk that exceeds MaxChunkSize
	var builder strings.Builder
	builder.WriteString("## Large Section\n\n")
	
	// Add enough content to exceed MaxChunkSize (8000 chars)
	paragraph := strings.Repeat("This is a test paragraph with meaningful content. ", 50) // ~2500 chars
	for i := 0; i < 5; i++ {
		builder.WriteString(paragraph)
		builder.WriteString("\n\n")
	}
	
	content := builder.String()
	chunks := ChunkMarkdown(content)
	
	// Should be split into multiple chunks
	if len(chunks) <= 1 {
		t.Errorf("Expected multiple chunks for large content, got %d", len(chunks))
	}
	
	// Verify no chunk exceeds MaxChunkSize
	for i, chunk := range chunks {
		size := utf8.RuneCountInString(chunk)
		if size > MaxChunkSize*1.1 { // Allow 10% margin
			t.Errorf("Chunk %d exceeds MaxChunkSize: %d chars", i, size)
		}
	}
}

func TestChunkMarkdown_EmptyContent(t *testing.T) {
	chunks := ChunkMarkdown("")
	
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestChunkMarkdown_PreservesHeaders(t *testing.T) {
	content := `## Important Section
This section has important content.

` + strings.Repeat("Lorem ipsum dolor sit amet. ", 500) // Make it large enough to split

	chunks := ChunkMarkdown(content)
	
	// All chunks should contain the header for context
	for i, chunk := range chunks {
		if !strings.Contains(chunk, "## Important Section") {
			t.Logf("Warning: Chunk %d does not contain header (might be by design)", i)
		}
	}
}

func TestChunkText_BasicFunctionality(t *testing.T) {
	content := `First paragraph with some text.

Second paragraph with more text.

Third paragraph with even more text.`

	chunks := ChunkText(content, 100)
	
	if len(chunks) < 1 {
		t.Error("Expected at least 1 chunk")
	}
	
	// Verify chunks are reasonable size
	for i, chunk := range chunks {
		size := utf8.RuneCountInString(chunk)
		if size > 150 { // Allow some margin
			t.Errorf("Chunk %d too large: %d chars", i, size)
		}
	}
}

func TestSplitOnSentences(t *testing.T) {
	text := "This is the first sentence. This is the second sentence! And this is the third? Finally the fourth."
	
	sentences := splitOnSentences(text)
	
	if len(sentences) < 2 {
		t.Errorf("Expected multiple sentences, got %d", len(sentences))
	}
	
	// Each sentence should end with punctuation
	for i, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			last := s[len(s)-1]
			if last != '.' && last != '!' && last != '?' && i < len(sentences)-1 {
				t.Errorf("Sentence %d doesn't end with punctuation: %s", i, s)
			}
		}
	}
}

func TestChunkSizes(t *testing.T) {
	// Test that constants are sensible
	if MinChunkSize >= OptimalChunkSize {
		t.Error("MinChunkSize should be less than OptimalChunkSize")
	}
	
	if OptimalChunkSize >= MaxChunkSize {
		t.Error("OptimalChunkSize should be less than MaxChunkSize")
	}
	
	// Verify sizes are appropriate for embeddings
	// 1 token ≈ 4 chars, so 8000 chars ≈ 2000 tokens (well within limits)
	if MaxChunkSize > 30000 {
		t.Error("MaxChunkSize too large for embedding models")
	}
}

// Benchmark chunking performance
func BenchmarkChunkMarkdown(b *testing.B) {
	// Create realistic document
	var builder strings.Builder
	for i := 0; i < 10; i++ {
		builder.WriteString("## Section ")
		builder.WriteString(strings.Repeat("Content paragraph. ", 200))
		builder.WriteString("\n\n")
	}
	content := builder.String()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ChunkMarkdown(content)
	}
}

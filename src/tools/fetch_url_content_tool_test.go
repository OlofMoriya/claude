package tools

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchURLContentToolRun_MarkdownSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
			<html>
			  <head><title>Test Article</title></head>
			  <body>
			    <nav>
			      <a href="/docs">Docs</a>
			      <a href="https://example.org/blog">Blog</a>
			    </nav>
			    <article>
			      <h1>Welcome</h1>
			      <p>This is useful content.</p>
			    </article>
			  </body>
			</html>
		`))
	}))
	defer server.Close()

	tool := &FetchURLContentTool{client: &http.Client{Timeout: 5 * time.Second}}
	out, err := tool.Run(map[string]string{"url": server.URL})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out, "Welcome") {
		t.Fatalf("expected extracted content, got: %s", out)
	}
	if !strings.Contains(out, "This is useful content.") {
		t.Fatalf("expected body content, got: %s", out)
	}
	if !strings.Contains(out, "## Navigation") {
		t.Fatalf("expected navigation section, got: %s", out)
	}
	if !strings.Contains(out, "Docs -> ") || !strings.Contains(out, "Blog -> https://example.org/blog") {
		t.Fatalf("expected navigation links, got: %s", out)
	}
}

func TestFetchURLContentToolRun_MarkdownOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><nav><a href="/home">Home</a></nav><article><p>Plain text body.</p></article></body></html>`))
	}))
	defer server.Close()

	tool := &FetchURLContentTool{client: &http.Client{Timeout: 5 * time.Second}}
	out, err := tool.Run(map[string]string{"url": server.URL})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out, "Plain text body.") {
		t.Fatalf("expected markdown content, got: %s", out)
	}
	if !strings.Contains(out, "## Navigation") || !strings.Contains(out, "Home -> ") {
		t.Fatalf("expected navigation section, got: %s", out)
	}
}

func TestFetchURLContentToolRun_TruncatesContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Truncate</title></head><body><article><p>abcdefghijklmnopqrstuvwxyz</p></article></body></html>`))
	}))
	defer server.Close()

	tool := &FetchURLContentTool{client: &http.Client{Timeout: 5 * time.Second}}
	out, err := tool.Run(map[string]string{
		"url":       server.URL,
		"max_chars": "10",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out, "_Truncated at 10 characters._") {
		t.Fatalf("expected truncation notice, got: %s", out)
	}
	if !strings.Contains(out, "...[truncated]") {
		t.Fatalf("expected truncation marker, got: %s", out)
	}
}

func TestFetchURLContentToolRun_InvalidScheme(t *testing.T) {
	tool := &FetchURLContentTool{}
	_, err := tool.Run(map[string]string{"url": "file:///etc/passwd"})
	if err == nil || !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("expected http/https error, got: %v", err)
	}
}

func TestFetchURLContentToolRun_NonHTMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	tool := &FetchURLContentTool{client: &http.Client{Timeout: 5 * time.Second}}
	_, err := tool.Run(map[string]string{"url": server.URL})
	if err == nil || !strings.Contains(err.Error(), "html content") {
		t.Fatalf("expected non-html error, got: %v", err)
	}
}

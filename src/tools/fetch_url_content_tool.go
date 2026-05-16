package tools

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"owl/data"
	"strconv"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"golang.org/x/net/html"
)

const (
	fetchURLContentTimeout      = 15 * time.Second
	fetchURLContentMaxBodyBytes = 2 * 1024 * 1024
	fetchURLContentDefaultChars = 12000
	fetchURLContentMaxChars     = 30000
	fetchURLContentNavMaxItems  = 12
)

type FetchURLContentTool struct {
	client *http.Client
}

func (tool *FetchURLContentTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {}

func (tool *FetchURLContentTool) GetName() string {
	return "fetch_url_content"
}

func (tool *FetchURLContentTool) GetGroups() []ToolGroup {
	return []ToolGroup{ToolGroupPlanner, ToolGroupDeveloper, ToolGroupManager, ToolGroupSecretary}
}

func (tool *FetchURLContentTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:         tool.GetName(),
		Description:  "Fetches a webpage by URL and returns main content as markdown, not full raw HTML.",
		Groups:       []ToolGroup{ToolGroupPlanner, ToolGroupDeveloper, ToolGroupManager, ToolGroupSecretary},
		Dependencies: []ToolDependency{},
		InputSchema: InputSchema{
			Type:     "object",
			Required: []string{"url"},
			Properties: map[string]Property{
				"url": {
					Type:        "string",
					Description: "HTTP/HTTPS URL to fetch.",
				},
				"max_chars": {
					Type:        "integer",
					Description: "Maximum number of characters to return. Default 12000, max 30000.",
				},
			},
		},
	}, REMOTE
}

func (tool *FetchURLContentTool) Run(i map[string]string) (string, error) {
	inputURL := getInputValue(i, "url", "URL")
	inputURL = strings.TrimSpace(inputURL)
	if inputURL == "" {
		return "", fmt.Errorf("url is required")
	}

	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return "", fmt.Errorf("invalid url")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("url must use http or https")
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid url")
	}

	maxChars := fetchURLContentDefaultChars
	if rawMax := strings.TrimSpace(getInputValue(i, "max_chars", "MaxChars")); rawMax != "" {
		parsedMax, parseErr := strconv.Atoi(rawMax)
		if parseErr != nil {
			return "", fmt.Errorf("max_chars must be an integer")
		}
		if parsedMax < 1 {
			return "", fmt.Errorf("max_chars must be >= 1")
		}
		if parsedMax > fetchURLContentMaxChars {
			parsedMax = fetchURLContentMaxChars
		}
		maxChars = parsedMax
	}

	client := tool.client
	if client == nil {
		client = &http.Client{Timeout: fetchURLContentTimeout}
	}

	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; owl-fetch-url-content/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if contentType != "" && !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml+xml") {
		return "", fmt.Errorf("url did not return html content")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchURLContentMaxBodyBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if len(body) > fetchURLContentMaxBodyBytes {
		return "", fmt.Errorf("response body too large")
	}

	article, err := readability.FromReader(strings.NewReader(string(body)), parsedURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract readable content: %w", err)
	}

	var htmlBuffer bytes.Buffer
	if renderErr := article.RenderHTML(&htmlBuffer); renderErr != nil {
		return "", fmt.Errorf("failed to render extracted content: %w", renderErr)
	}

	content := strings.TrimSpace(htmlBuffer.String())
	if content == "" {
		return "", fmt.Errorf("no readable content found")
	}

	markdown, convErr := htmltomarkdown.ConvertString(content)
	if convErr != nil {
		return "", fmt.Errorf("failed to convert content to markdown: %w", convErr)
	}
	content = strings.TrimSpace(markdown)

	content, truncated := truncateContent(content, maxChars)
	navigationSection := buildNavigationSection(body, parsedURL)

	if truncated {
		content += fmt.Sprintf("\n\n_Truncated at %d characters._", maxChars)
	}

	return content + "\n\n" + navigationSection, nil
}

func buildNavigationSection(htmlBody []byte, baseURL *url.URL) string {
	links := extractNavigationLinks(htmlBody, baseURL, fetchURLContentNavMaxItems)
	if len(links) == 0 {
		return "## Navigation\n- None found"
	}

	lines := []string{"## Navigation"}
	for _, link := range links {
		lines = append(lines, fmt.Sprintf("- %s -> %s", link.Text, link.URL))
	}

	return strings.Join(lines, "\n")
}

type navLink struct {
	Text string
	URL  string
}

func extractNavigationLinks(htmlBody []byte, baseURL *url.URL, maxItems int) []navLink {
	root, err := html.Parse(bytes.NewReader(htmlBody))
	if err != nil {
		return nil
	}

	regions := findNavigationRegions(root)
	if len(regions) == 0 {
		return nil
	}

	links := make([]navLink, 0, maxItems)
	seen := map[string]bool{}
	for _, region := range regions {
		collectLinks(region, baseURL, seen, &links, maxItems)
		if len(links) >= maxItems {
			break
		}
	}

	return links
}

func findNavigationRegions(root *html.Node) []*html.Node {
	regions := []*html.Node{}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if tag == "nav" || hasNavigationRole(n) || hasNavigationClassOrID(n) || tag == "header" {
				regions = append(regions, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return regions
}

func collectLinks(region *html.Node, baseURL *url.URL, seen map[string]bool, out *[]navLink, maxItems int) {
	if len(*out) >= maxItems {
		return
	}

	if region.Type == html.ElementNode && strings.EqualFold(region.Data, "a") {
		href := strings.TrimSpace(getHTMLAttr(region, "href"))
		if href != "" && !isIgnoredHref(href) {
			if resolved, ok := resolveLink(baseURL, href); ok {
				if !seen[resolved] {
					text := strings.TrimSpace(normalizeWhitespace(extractNodeText(region)))
					if text != "" {
						seen[resolved] = true
						*out = append(*out, navLink{Text: text, URL: resolved})
					}
				}
			}
		}
	}

	for c := region.FirstChild; c != nil; c = c.NextSibling {
		collectLinks(c, baseURL, seen, out, maxItems)
		if len(*out) >= maxItems {
			return
		}
	}
}

func hasNavigationRole(n *html.Node) bool {
	role := strings.ToLower(strings.TrimSpace(getHTMLAttr(n, "role")))
	return role == "navigation"
}

func hasNavigationClassOrID(n *html.Node) bool {
	className := strings.ToLower(getHTMLAttr(n, "class"))
	id := strings.ToLower(getHTMLAttr(n, "id"))
	keywords := []string{"nav", "menu", "navbar", "main-nav", "site-nav", "top-nav", "header-nav"}
	for _, keyword := range keywords {
		if strings.Contains(className, keyword) || strings.Contains(id, keyword) {
			return true
		}
	}
	return false
}

func getHTMLAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func resolveLink(baseURL *url.URL, href string) (string, bool) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	if baseURL == nil {
		return strings.TrimSpace(parsed.String()), true
	}
	resolved := baseURL.ResolveReference(parsed)
	if resolved == nil {
		return "", false
	}
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}
	return strings.TrimSpace(resolved.String()), true
}

func isIgnoredHref(href string) bool {
	lower := strings.ToLower(strings.TrimSpace(href))
	return lower == "" || lower == "#" || strings.HasPrefix(lower, "javascript:")
}

func extractNodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			b.WriteString(" ")
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func truncateContent(content string, maxChars int) (string, bool) {
	runes := []rune(content)
	if len(runes) <= maxChars {
		return content, false
	}
	return strings.TrimSpace(string(runes[:maxChars])) + "\n\n...[truncated]", true
}

func getInputValue(input map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := input[key]; ok {
			return value
		}
	}
	return ""
}

func init() {
	Register(&FetchURLContentTool{})
}

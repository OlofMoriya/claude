package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"owl/data"
	"owl/logger"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	stockModeSingle       = "single"
	stockModeCustomList   = "custom_list"
	stockModeGeneral      = "general_market"
	stockModeMyStandards  = "my_standards"
	defaultStocksTimeout  = 12 * time.Second
	yahooChartEndpointURL = "https://query1.finance.yahoo.com/v8/finance/chart"
)

var generalMarketGroups = []struct {
	Name    string
	Symbols []string
}{
	{Name: "global", Symbols: []string{"VT", "ACWI", "URTH"}},
	{Name: "swedish", Symbols: []string{"^OMXS30", "EQT.ST", "ATCO-A.ST", "VOLV-B.ST"}},
	{Name: "american", Symbols: []string{"^GSPC", "^IXIC", "^DJI", "^RUT"}},
	{Name: "asian", Symbols: []string{"^N225", "^HSI", "000001.SS", "^KS11"}},
}

var myStandardSymbols = []string{"AAPL", "TSLA", "MSFT", "MU", "IBM", "GOOGL", "META", "GME"}

type StockPriceLookupTool struct {
	client  *http.Client
	baseURL string
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol                string  `json:"symbol"`
				ShortName             string  `json:"shortName"`
				LongName              string  `json:"longName"`
				RegularMarketPrice    float64 `json:"regularMarketPrice"`
				RegularMarketTimeUnix int64   `json:"regularMarketTime"`
				ChartPreviousClose    float64 `json:"chartPreviousClose"`
				PreviousClose         float64 `json:"previousClose"`
				Currency              string  `json:"currency"`
				ExchangeName          string  `json:"exchangeName"`
				CurrentTradingPeriod  struct {
					Regular struct {
						Start int64 `json:"start"`
						End   int64 `json:"end"`
					} `json:"regular"`
				} `json:"currentTradingPeriod"`
			} `json:"meta"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

type stockQuote struct {
	Symbol      string
	Name        string
	Price       float64
	Change      float64
	ChangePct   float64
	Currency    string
	MarketState string
	MarketTime  time.Time
}

func (tool *StockPriceLookupTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {}

func (tool *StockPriceLookupTool) GetName() string {
	return "stock_price_lookup"
}

func (tool *StockPriceLookupTool) GetGroups() []string {
	return []string{"manage"}
}

func (tool *StockPriceLookupTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:        tool.GetName(),
		Description: "Fetches current stock and index prices from Yahoo Finance (no API key). Supports single symbol lookup, custom lists, a general multi-region market preset, and a personal standards preset.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Mode": {
					Type:        "string",
					Description: "Lookup mode: single, custom_list, general_market, or my_standards. Default is single if Symbol is provided.",
				},
				"Symbol": {
					Type:        "string",
					Description: "Ticker symbol for single mode (examples: AAPL, TSLA, ^OMXS30, EQT.ST).",
				},
				"Symbols": {
					Type:        "string",
					Description: "Comma-separated symbol list for custom_list mode.",
				},
			},
		},
	}, LOCAL
}

func (tool *StockPriceLookupTool) Run(i map[string]string) (string, error) {
	mode := normalizeMode(i["Mode"], i["Symbol"])
	stockDebugf("stock_price_lookup: mode=%s input=%v", mode, i)

	lookupGroups, err := symbolsByMode(mode, i["Symbol"], i["Symbols"])
	if err != nil {
		stockDebugf("stock_price_lookup: symbolsByMode failed: %v", err)
		return "", err
	}

	flatSymbols := flattenGroupSymbols(lookupGroups)
	stockDebugf("stock_price_lookup: resolved symbols=%v", flatSymbols)
	quotesBySymbol, err := tool.fetchQuotes(flatSymbols)
	if err != nil {
		stockDebugf("stock_price_lookup: fetchQuotes failed: %v", err)
		return "", err
	}

	return formatStockOutput(mode, lookupGroups, quotesBySymbol), nil
}

func normalizeMode(rawMode, rawSymbol string) string {
	mode := strings.ToLower(strings.TrimSpace(rawMode))
	if mode != "" {
		return mode
	}
	if strings.TrimSpace(rawSymbol) != "" {
		return stockModeSingle
	}
	return stockModeMyStandards
}

func symbolsByMode(mode, symbol, symbolsList string) ([]stockGroup, error) {
	switch mode {
	case stockModeSingle:
		s := normalizeSymbol(symbol)
		if s == "" {
			return nil, fmt.Errorf("Symbol is required in single mode")
		}
		return []stockGroup{{Name: "single", Symbols: []string{s}}}, nil
	case stockModeCustomList:
		symbols := parseSymbolsCSV(symbolsList)
		if len(symbols) == 0 {
			return nil, fmt.Errorf("Symbols is required in custom_list mode")
		}
		return []stockGroup{{Name: "custom", Symbols: symbols}}, nil
	case stockModeGeneral:
		groups := make([]stockGroup, 0, len(generalMarketGroups))
		for _, group := range generalMarketGroups {
			groups = append(groups, stockGroup{Name: group.Name, Symbols: append([]string{}, group.Symbols...)})
		}
		return groups, nil
	case stockModeMyStandards:
		return []stockGroup{{Name: "my_standards", Symbols: append([]string{}, myStandardSymbols...)}}, nil
	default:
		return nil, fmt.Errorf("invalid Mode '%s'. valid modes: single, custom_list, general_market, my_standards", mode)
	}
}

type stockGroup struct {
	Name    string
	Symbols []string
}

func flattenGroupSymbols(groups []stockGroup) []string {
	seen := map[string]struct{}{}
	all := make([]string, 0)
	for _, group := range groups {
		for _, symbol := range group.Symbols {
			if _, exists := seen[symbol]; exists {
				continue
			}
			seen[symbol] = struct{}{}
			all = append(all, symbol)
		}
	}
	sort.Strings(all)
	return all
}

func normalizeSymbol(s string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(s))
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func parseSymbolsCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		symbol := normalizeSymbol(part)
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	return out
}

func (tool *StockPriceLookupTool) fetchQuotes(symbols []string) (map[string]stockQuote, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("no symbols to fetch")
	}

	baseURL := strings.TrimSpace(tool.baseURL)
	if baseURL == "" {
		baseURL = yahooChartEndpointURL
	}

	client := tool.client
	if client == nil {
		client = &http.Client{Timeout: defaultStocksTimeout}
	}

	_, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid quote endpoint: %w", err)
	}
	results := make(map[string]stockQuote, len(symbols))
	var firstErr error

	for _, symbol := range symbols {
		quote, err := fetchChartQuote(client, baseURL, symbol)
		if err != nil {
			stockDebugf("stock_price_lookup: symbol=%s fetch failed: %v", symbol, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		results[symbol] = quote
	}

	if len(results) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

func fetchChartQuote(client *http.Client, baseURL, symbol string) (stockQuote, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return stockQuote{}, fmt.Errorf("invalid chart endpoint: %w", err)
	}
	u.Path = path.Join(u.Path, url.PathEscape(symbol))
	q := u.Query()
	q.Set("interval", "1d")
	q.Set("range", "1d")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return stockQuote{}, fmt.Errorf("failed to build chart request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; owl-stock-tool/1.0)")

	stockDebugf("stock_price_lookup: requesting %s", u.String())

	resp, err := client.Do(req)
	if err != nil {
		return stockQuote{}, fmt.Errorf("failed to fetch %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return stockQuote{}, fmt.Errorf("failed to read chart response body for %s: %w", symbol, err)
	}
	stockDebugf("stock_price_lookup: symbol=%s response status=%d body_len=%d", symbol, resp.StatusCode, len(bodyBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		stockDebugf("stock_price_lookup: symbol=%s non-2xx body=%s", symbol, truncateForLog(string(bodyBytes), 600))
		return stockQuote{}, fmt.Errorf("quote endpoint returned status %d for %s", resp.StatusCode, symbol)
	}

	var payload yahooChartResponse
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		stockDebugf("stock_price_lookup: symbol=%s JSON parse error=%v body=%s", symbol, err, truncateForLog(string(bodyBytes), 600))
		return stockQuote{}, fmt.Errorf("failed to parse quote response for %s: %w", symbol, err)
	}

	if len(payload.Chart.Result) == 0 {
		return stockQuote{}, fmt.Errorf("no chart result returned for %s", symbol)
	}

	meta := payload.Chart.Result[0].Meta
	name := strings.TrimSpace(meta.ShortName)
	if name == "" {
		name = strings.TrimSpace(meta.LongName)
	}
	price := meta.RegularMarketPrice
	previousClose := meta.ChartPreviousClose
	if previousClose == 0 {
		previousClose = meta.PreviousClose
	}

	change := 0.0
	changePct := 0.0
	if previousClose != 0 {
		change = price - previousClose
		changePct = (change / previousClose) * 100
	}

	marketState := "UNKNOWN"
	now := time.Now().UTC().Unix()
	if meta.CurrentTradingPeriod.Regular.Start > 0 && meta.CurrentTradingPeriod.Regular.End > 0 {
		if now >= meta.CurrentTradingPeriod.Regular.Start && now <= meta.CurrentTradingPeriod.Regular.End {
			marketState = "REGULAR"
		} else if now < meta.CurrentTradingPeriod.Regular.Start {
			marketState = "PRE"
		} else {
			marketState = "CLOSED"
		}
	}

	return stockQuote{
		Symbol:      strings.ToUpper(meta.Symbol),
		Name:        name,
		Price:       price,
		Change:      change,
		ChangePct:   changePct,
		Currency:    meta.Currency,
		MarketState: marketState,
		MarketTime:  time.Unix(meta.RegularMarketTimeUnix, 0).UTC(),
	}, nil
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func stockDebugf(format string, args ...interface{}) {
	if logger.Debug == nil {
		return
	}
	logger.Debug.Printf(format, args...)
}

func formatStockOutput(mode string, groups []stockGroup, quotes map[string]stockQuote) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Stock lookup mode: %s\n", mode))

	for _, group := range groups {
		b.WriteString(fmt.Sprintf("\n[%s]\n", group.Name))
		for _, symbol := range group.Symbols {
			quote, ok := quotes[symbol]
			if !ok {
				b.WriteString(fmt.Sprintf("- %s: no quote returned\n", symbol))
				continue
			}

			timeString := "n/a"
			if !quote.MarketTime.IsZero() {
				timeString = quote.MarketTime.Format(time.RFC3339)
			}

			namePart := ""
			if strings.TrimSpace(quote.Name) != "" {
				namePart = fmt.Sprintf(" (%s)", quote.Name)
			}

			b.WriteString(fmt.Sprintf(
				"- %s%s: %.4f %s (%.4f / %.2f%%), state=%s, time=%s\n",
				quote.Symbol,
				namePart,
				quote.Price,
				emptyToNA(quote.Currency),
				quote.Change,
				quote.ChangePct,
				emptyToNA(quote.MarketState),
				timeString,
			))
		}
	}

	return strings.TrimSpace(b.String())
}

func emptyToNA(s string) string {
	if strings.TrimSpace(s) == "" {
		return "N/A"
	}
	return s
}

func init() {
	Register(&StockPriceLookupTool{})
}

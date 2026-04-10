package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
)

func TestSymbolsByModeSingleRequiresSymbol(t *testing.T) {
	_, err := symbolsByMode(stockModeSingle, "", "")
	if err == nil {
		t.Fatalf("expected error when symbol is missing")
	}
}

func TestSymbolsByModeGeneralContainsRegions(t *testing.T) {
	groups, err := symbolsByMode(stockModeGeneral, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(groups))
	}
	if groups[0].Name != "global" || groups[1].Name != "swedish" || groups[2].Name != "american" || groups[3].Name != "asian" {
		t.Fatalf("unexpected group names: %+v", groups)
	}
}

func TestRunMyStandardsIncludesConfiguredTickers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := path.Base(r.URL.Path)
		decoded, _ := url.PathUnescape(symbol)
		fmt.Fprintf(w, `{"chart":{"result":[{"meta":{"symbol":"%s","shortName":"%s","regularMarketPrice":100,"chartPreviousClose":99,"currency":"USD","regularMarketTime":1710000000,"currentTradingPeriod":{"regular":{"start":1709990000,"end":1710010000}}}}],"error":null}}`, decoded, decoded)
	}))
	defer server.Close()

	tool := &StockPriceLookupTool{
		client:  server.Client(),
		baseURL: server.URL,
	}

	output, err := tool.Run(map[string]string{"Mode": stockModeMyStandards})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, symbol := range myStandardSymbols {
		if !strings.Contains(output, symbol) {
			t.Fatalf("expected output to include symbol %s, got: %s", symbol, output)
		}
	}
}

func TestRunCustomListDeduplicatesSymbols(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		fmt.Fprint(w, `{"chart":{"result":[{"meta":{"symbol":"AAPL","shortName":"Apple","regularMarketPrice":1,"chartPreviousClose":0.9,"currency":"USD","regularMarketTime":1710000000,"currentTradingPeriod":{"regular":{"start":1709990000,"end":1710010000}}}}],"error":null}}`)
	}))
	defer server.Close()

	tool := &StockPriceLookupTool{client: server.Client(), baseURL: server.URL}
	_, err := tool.Run(map[string]string{"Mode": stockModeCustomList, "Symbols": "aapl, AAPL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected one request for deduped symbols, got %d", requestCount)
	}
}

func TestFetchQuotesHandlesEndpointFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	}))
	defer server.Close()

	tool := &StockPriceLookupTool{client: server.Client(), baseURL: server.URL}
	_, err := tool.fetchQuotes([]string{"AAPL"})
	if err == nil {
		t.Fatalf("expected endpoint error")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// yahooSource is the fallback for Indian stocks via Yahoo Finance's chart API
// (keyless). Expects a Yahoo symbol, e.g. "RELIANCE.NS".
type yahooSource struct{ client *http.Client }

func (p yahooSource) ID() string { return "yahoo" }

func (p yahooSource) GetQuote(ctx context.Context, symbol string) (Quote, error) {
	u := "https://query1.finance.yahoo.com/v8/finance/chart/" +
		url.PathEscape(symbol) + "?interval=1d&range=1d"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", nseUA)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("yahoo: status %d", resp.StatusCode)
	}

	var out struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					Currency           string  `json:"currency"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Quote{}, err
	}
	if len(out.Chart.Result) == 0 || out.Chart.Result[0].Meta.RegularMarketPrice == 0 {
		return Quote{}, fmt.Errorf("yahoo: no price for %q", symbol)
	}
	m := out.Chart.Result[0].Meta
	cur := m.Currency
	if cur == "" {
		cur = "INR"
	}
	return Quote{Price: m.RegularMarketPrice, Currency: cur, At: time.Now().UTC()}, nil
}

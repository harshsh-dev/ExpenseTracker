package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// coinGecko fetches crypto prices directly in INR (no API key required).
// Symbol is a CoinGecko coin id, e.g. "bitcoin", "ethereum".
type coinGecko struct{ client *http.Client }

func (c coinGecko) ID() string { return "coingecko" }

func (c coinGecko) GetQuote(ctx context.Context, symbol string) (Quote, error) {
	id := strings.ToLower(strings.TrimSpace(symbol))
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=inr", id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("coingecko: status %d", resp.StatusCode)
	}

	var out map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Quote{}, err
	}
	price, ok := out[id]["inr"]
	if !ok || price == 0 {
		return Quote{}, fmt.Errorf("coingecko: no price for %q", id)
	}
	return Quote{Price: price, Currency: "INR", At: time.Now().UTC()}, nil
}

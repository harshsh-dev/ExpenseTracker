package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// mfAPI fetches Indian mutual fund NAVs from MFAPI.in (AMFI data, no key).
// Symbol is the AMFI scheme code, e.g. "120503". NAV updates once per day.
type mfAPI struct{ client *http.Client }

func (m mfAPI) ID() string { return "mfapi" }

func (m mfAPI) GetQuote(ctx context.Context, symbol string) (Quote, error) {
	u := fmt.Sprintf("https://api.mfapi.in/mf/%s/latest", url.PathEscape(symbol))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("mfapi: status %d", resp.StatusCode)
	}

	var out struct {
		Status string `json:"status"`
		Data   []struct {
			Date string `json:"date"`
			NAV  string `json:"nav"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Quote{}, err
	}
	if len(out.Data) == 0 {
		return Quote{}, fmt.Errorf("mfapi: no NAV for scheme %q", symbol)
	}
	price, err := strconv.ParseFloat(out.Data[0].NAV, 64)
	if err != nil {
		return Quote{}, fmt.Errorf("mfapi: bad NAV %q", out.Data[0].NAV)
	}
	at := time.Now().UTC()
	if d, err := time.Parse("02-01-2006", out.Data[0].Date); err == nil {
		at = d
	}
	return Quote{Price: price, Currency: "INR", At: at}, nil
}

// searchMF resolves a query to scheme code + name via MFAPI's search.
func searchMF(ctx context.Context, client *http.Client, q string) ([]SearchHit, error) {
	u := "https://api.mfapi.in/mf/search?q=" + url.QueryEscape(q)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mfapi search: status %d", resp.StatusCode)
	}
	var out []struct {
		SchemeCode json.Number `json:"schemeCode"`
		SchemeName string      `json:"schemeName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	hits := make([]SearchHit, 0, len(out))
	for _, o := range out {
		hits = append(hits, SearchHit{Symbol: o.SchemeCode.String(), Name: o.SchemeName})
	}
	return hits, nil
}

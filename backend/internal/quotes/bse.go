package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BSE's JSON APIs are keyless but reject non-browser callers, so we send
// browser-like headers plus the bseindia.com Origin/Referer. Unlike NSE, no
// cookie bootstrap is required.
const (
	bseQuoteURL = "https://api.bseindia.com/BseIndiaAPI/api/getScripHeaderData/w"
	bseListURL  = "https://api.bseindia.com/BseIndiaAPI/api/ListofScripData/w" +
		"?Group=&Scripcode=&industry=&segment=Equity&status=Active"
)

func bseHeaders(req *http.Request) {
	req.Header.Set("User-Agent", nseUA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.bseindia.com/")
	req.Header.Set("Origin", "https://www.bseindia.com")
}

// bseEntry is one row of the BSE equity master list. Quotes are keyed by the
// numeric scrip code; the ticker and name are kept for display and search.
type bseEntry struct {
	Code   string // scrip code, used for quotes (e.g. "500180")
	Ticker string // scrip_id (e.g. "HDFCBANK")
	Name   string
	lc     string // lowercase "ticker name"
	lcNoSp string // lowercase, spaces stripped
}

// bseDir is a shared, TTL-cached copy of the BSE active-equity master list.
type bseDir struct {
	mu       sync.Mutex
	entries  []bseEntry
	loadedAt time.Time
}

func (d *bseDir) ensure(ctx context.Context, client *http.Client) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.entries) > 0 && time.Since(d.loadedAt) < 12*time.Hour {
		return nil
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, bseListURL, nil)
	bseHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bse: scrip list status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rows []struct {
		ScripCD    string `json:"SCRIP_CD"`
		ScripID    string `json:"scrip_id"`
		ScripName  string `json:"Scrip_Name"`
		IssuerName string `json:"Issuer_Name"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return fmt.Errorf("bse: scrip list decode: %w", err)
	}

	entries := make([]bseEntry, 0, len(rows))
	for _, r := range rows {
		code := strings.TrimSpace(r.ScripCD)
		ticker := strings.TrimSpace(r.ScripID)
		if code == "" {
			continue
		}
		name := strings.TrimSpace(r.IssuerName)
		if name == "" {
			name = strings.TrimSpace(r.ScripName)
		}
		lc := strings.ToLower(ticker + " " + name)
		entries = append(entries, bseEntry{
			Code:   code,
			Ticker: ticker,
			Name:   name,
			lc:     lc,
			lcNoSp: strings.ReplaceAll(lc, " ", ""),
		})
	}
	if len(entries) == 0 {
		return fmt.Errorf("bse: empty scrip list")
	}
	d.entries = entries
	d.loadedAt = time.Now()
	return nil
}

// bseSource implements Provider for BSE equities. The symbol it expects is the
// numeric BSE scrip code (e.g. "500180" for HDFCBANK).
type bseSource struct {
	client *http.Client
	dir    *bseDir
}

func (p bseSource) ID() string { return "bse" }

func (p bseSource) GetQuote(ctx context.Context, symbol string) (Quote, error) {
	code := strings.TrimSpace(symbol)
	u := bseQuoteURL + "?Debtflag=&scripcode=" + url.QueryEscape(code) + "&seriesid="
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	bseHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("bse: status %d", resp.StatusCode)
	}

	var out struct {
		CurrRate struct {
			LTP string `json:"LTP"`
		} `json:"CurrRate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Quote{}, fmt.Errorf("bse: decode: %w", err)
	}
	price, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(out.CurrRate.LTP), ",", ""), 64)
	if err != nil || price == 0 {
		return Quote{}, fmt.Errorf("bse: no price for %q", code)
	}
	return Quote{Price: price, Currency: "INR", At: time.Now().UTC()}, nil
}

// search resolves a query to BSE scrip codes by ranked matching against the
// cached scrip master list. Returned hits use the scrip code as the symbol
// (what GetQuote needs) and show the company name + ticker.
func (p bseSource) search(ctx context.Context, q string) ([]SearchHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if err := p.dir.ensure(ctx, p.client); err != nil {
		return nil, err
	}

	lc := strings.ToLower(q)
	noSp := strings.ReplaceAll(lc, " ", "")
	tokens := strings.Fields(lc)

	p.dir.mu.Lock()
	entries := p.dir.entries
	p.dir.mu.Unlock()

	type scored struct {
		hit  SearchHit
		tier int
		span int
	}
	out := make([]scored, 0, 16)
	for _, e := range entries {
		tickLC := strings.ToLower(e.Ticker)
		tier, span := -1, 0
		switch {
		case tickLC == lc || e.Code == q:
			tier = 0
		case strings.HasPrefix(tickLC, noSp):
			tier = 1
		case strings.Contains(e.lcNoSp, noSp):
			tier = 2
		case allTokensPresent(e.lc, tokens):
			tier = 3
		default:
			if s, ok := subsequenceSpan(e.lcNoSp, noSp); ok {
				tier, span = 4, s
			}
		}
		if tier == -1 {
			continue
		}
		name := e.Name
		if e.Ticker != "" {
			name = fmt.Sprintf("%s (%s)", e.Name, e.Ticker)
		}
		out = append(out, scored{SearchHit{Symbol: e.Code, Name: name}, tier, span})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].tier != out[j].tier {
			return out[i].tier < out[j].tier
		}
		if out[i].span != out[j].span {
			return out[i].span < out[j].span
		}
		return len(out[i].hit.Name) < len(out[j].hit.Name)
	})

	const limit = 15
	if len(out) > limit {
		out = out[:limit]
	}
	hits := make([]SearchHit, len(out))
	for i, s := range out {
		hits[i] = s.hit
	}
	return hits, nil
}

// bseProvider serves BSE stocks: the BSE JSON API as the primary source, Yahoo
// Finance (".BO" suffix on the scrip code) as the fallback. Symbol is the
// numeric scrip code.
type bseProvider struct {
	bse   bseSource
	yahoo yahooSource
}

func (p bseProvider) ID() string { return "bse" }

func (p bseProvider) GetQuote(ctx context.Context, symbol string) (Quote, error) {
	if q, err := p.bse.GetQuote(ctx, symbol); err == nil && q.Price > 0 {
		return q, nil
	}
	ysym := strings.TrimSpace(symbol)
	if !strings.Contains(ysym, ".") {
		ysym += ".BO"
	}
	return p.yahoo.GetQuote(ctx, ysym)
}

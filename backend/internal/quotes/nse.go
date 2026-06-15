package quotes

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const nseUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// nseSession holds a cookie jar primed against nseindia.com. NSE blocks
// non-browser clients, so we spoof browser headers and bootstrap cookies by
// visiting the site before calling its JSON API. Shared across all symbols.
type nseSession struct {
	mu       sync.Mutex
	client   *http.Client
	booted   bool
	bootedAt time.Time
}

func newNSESession() *nseSession {
	jar, _ := cookiejar.New(nil)
	return &nseSession{client: &http.Client{Timeout: 12 * time.Second, Jar: jar}}
}

func (n *nseSession) setHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", nseUA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

// bootstrap visits the homepage + a quotes page to obtain session cookies.
func (n *nseSession) bootstrap(ctx context.Context) error {
	for _, u := range []string{
		"https://www.nseindia.com/",
		"https://www.nseindia.com/get-quotes/equity?symbol=RELIANCE",
	} {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		n.setHeaders(req, "https://www.nseindia.com/")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		resp, err := n.client.Do(req)
		if err != nil {
			return err
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	n.booted = true
	n.bootedAt = time.Now()
	return nil
}

// get performs an authenticated GET, bootstrapping/refreshing the session as
// needed (cookies are re-primed if stale or on a 401/403).
func (n *nseSession) get(ctx context.Context, apiURL, referer string) ([]byte, error) {
	n.mu.Lock()
	if !n.booted || time.Since(n.bootedAt) > 10*time.Minute {
		if err := n.bootstrap(ctx); err != nil {
			n.mu.Unlock()
			return nil, err
		}
	}
	n.mu.Unlock()

	body, status, err := n.do(ctx, apiURL, referer)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		n.mu.Lock()
		_ = n.bootstrap(ctx)
		n.mu.Unlock()
		body, status, err = n.do(ctx, apiURL, referer)
		if err != nil {
			return nil, err
		}
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("nse: status %d", status)
	}
	return body, nil
}

func (n *nseSession) do(ctx context.Context, apiURL, referer string) ([]byte, int, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	n.setHeaders(req, referer)
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// nseSource implements Provider for NSE equities. Symbol is the NSE ticker
// (e.g. "RELIANCE", "TCS"). dir is a shared, cached copy of the NSE equity
// master list used for symbol search.
type nseSource struct {
	session *nseSession
	dir     *equityDir
}

func (p nseSource) ID() string { return "nse" }

func (p nseSource) GetQuote(ctx context.Context, symbol string) (Quote, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	api := "https://www.nseindia.com/api/quote-equity?symbol=" + url.QueryEscape(sym)
	ref := "https://www.nseindia.com/get-quotes/equity?symbol=" + url.QueryEscape(sym)
	body, err := p.session.get(ctx, api, ref)
	if err != nil {
		return Quote{}, err
	}
	var out struct {
		PriceInfo struct {
			LastPrice float64 `json:"lastPrice"`
		} `json:"priceInfo"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return Quote{}, fmt.Errorf("nse: decode: %w", err)
	}
	if out.PriceInfo.LastPrice == 0 {
		return Quote{}, fmt.Errorf("nse: no price for %q", sym)
	}
	return Quote{Price: out.PriceInfo.LastPrice, Currency: "INR", At: time.Now().UTC()}, nil
}

// equityListURL is NSE's official, daily-updated master list of all listed
// equities (SYMBOL, NAME OF COMPANY, ...). We search against this locally
// because NSE's old /api/search/autocomplete endpoint was removed (returns 404).
const equityListURL = "https://nsearchives.nseindia.com/content/equities/EQUITY_L.csv"

// equityEntry is one row of the equity master list, pre-lowercased for search.
type equityEntry struct {
	Symbol string
	Name   string
	lc     string // lowercase "symbol name"
	lcNoSp string // lowercase, spaces stripped (matches glued queries)
}

// equityDir is a shared, TTL-cached copy of the NSE equity master list.
type equityDir struct {
	mu       sync.Mutex
	entries  []equityEntry
	loadedAt time.Time
}

// ensure (re)loads the equity list if it is missing or older than 12h.
func (d *equityDir) ensure(ctx context.Context, sess *nseSession) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.entries) > 0 && time.Since(d.loadedAt) < 12*time.Hour {
		return nil
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, equityListURL, nil)
	req.Header.Set("User-Agent", nseUA)
	req.Header.Set("Accept", "text/csv,*/*")
	resp, err := sess.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nse: equity list status %d", resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("nse: equity list parse: %w", err)
	}

	entries := make([]equityEntry, 0, len(rows))
	for i, row := range rows {
		if i == 0 || len(row) < 2 { // skip header / malformed
			continue
		}
		sym := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		if sym == "" {
			continue
		}
		lc := strings.ToLower(sym + " " + name)
		entries = append(entries, equityEntry{
			Symbol: sym,
			Name:   name,
			lc:     lc,
			lcNoSp: strings.ReplaceAll(lc, " ", ""),
		})
	}
	if len(entries) == 0 {
		return fmt.Errorf("nse: empty equity list")
	}
	d.entries = entries
	d.loadedAt = time.Now()
	return nil
}

// search resolves a query to NSE symbols by ranked matching against the cached
// equity master list. Tiers (best first): exact symbol, symbol prefix, glued
// substring, all-tokens-present, then a subsequence fallback for typo'd/glued
// names (e.g. "idfcbank" -> IDFCFIRSTB).
func (p nseSource) search(ctx context.Context, q string) ([]SearchHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if err := p.dir.ensure(ctx, p.session); err != nil {
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
		symLC := strings.ToLower(e.Symbol)
		tier, span := -1, 0
		switch {
		case symLC == lc:
			tier = 0
		case strings.HasPrefix(symLC, noSp):
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
		out = append(out, scored{SearchHit{Symbol: e.Symbol, Name: e.Name}, tier, span})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].tier != out[j].tier {
			return out[i].tier < out[j].tier
		}
		if out[i].span != out[j].span {
			return out[i].span < out[j].span
		}
		return len(out[i].hit.Symbol) < len(out[j].hit.Symbol)
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

// allTokensPresent reports whether every token is a substring of hay.
func allTokensPresent(hay string, tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	for _, t := range tokens {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	return true
}

// subsequenceSpan reports whether needle is a subsequence of hay and, if so,
// the span (last-first index) of the greedy match — smaller spans are tighter,
// more relevant matches.
func subsequenceSpan(hay, needle string) (int, bool) {
	if needle == "" {
		return 0, false
	}
	start, end, ni := -1, -1, 0
	for i := 0; i < len(hay) && ni < len(needle); i++ {
		if hay[i] == needle[ni] {
			if start == -1 {
				start = i
			}
			end = i
			ni++
		}
	}
	if ni == len(needle) {
		return end - start, true
	}
	return 0, false
}

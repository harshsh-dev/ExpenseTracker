package quotes

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"moneytracker/backend/internal/domain"
	"moneytracker/backend/internal/store"
)

// Service orchestrates price refreshes across providers, with caching and a
// single in-flight refresh at a time.
type Service struct {
	store     *store.Store
	providers map[string]Provider
	cache     *cache
	nse       nseSource
	bse       bseSource
	shared    *http.Client
	refreshMu sync.Mutex
}

// New wires the providers. No API keys are required (MFAPI, NSE, Yahoo and
// CoinGecko are all keyless).
func New(st *store.Store) *Service {
	shared := &http.Client{Timeout: 12 * time.Second}
	nse := nseSource{session: newNSESession(), dir: &equityDir{}}
	bse := bseSource{client: shared, dir: &bseDir{}}
	yahoo := yahooSource{client: shared}
	return &Service{
		store:  st,
		cache:  newCache(),
		nse:    nse,
		bse:    bse,
		shared: shared,
		providers: map[string]Provider{
			"coingecko": coinGecko{client: shared},
			"mfapi":     mfAPI{client: shared},
			"stock":     stockProvider{nse: nse, yahoo: yahoo},
			"bse":       bseProvider{bse: bse, yahoo: yahoo},
		},
	}
}

// SymbolResult reports the outcome of refreshing one investment.
type SymbolResult struct {
	ID     string  `json:"id"`
	Symbol string  `json:"symbol"`
	OK     bool    `json:"ok"`
	Price  float64 `json:"price,omitempty"`
	Error  string  `json:"error,omitempty"`
}

// RefreshResult is the response of a refresh run.
type RefreshResult struct {
	Investments []domain.Investment `json:"investments"`
	Results     []SymbolResult      `json:"results"`
	RefreshedAt time.Time           `json:"refreshedAt"`
}

// RefreshAll fetches prices for every investment that has an auto provider,
// a symbol and a quantity. Symbols are deduped via the TTL cache.
func (s *Service) RefreshAll(ctx context.Context) RefreshResult {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	results := []SymbolResult{}
	for _, inv := range s.store.ListInvestments() {
		if inv.Provider == "" || inv.Provider == "manual" || inv.Symbol == "" || inv.Quantity == nil {
			continue
		}
		p, ok := s.providers[inv.Provider]
		if !ok {
			results = append(results, SymbolResult{ID: inv.ID, Symbol: inv.Symbol,
				Error: "unknown provider: " + inv.Provider})
			continue
		}

		key := inv.Provider + ":" + inv.Symbol
		q, cached := s.cache.get(key)
		if !cached {
			fetched, err := p.GetQuote(ctx, inv.Symbol)
			if err != nil {
				results = append(results, SymbolResult{ID: inv.ID, Symbol: inv.Symbol, Error: err.Error()})
				continue
			}
			q = fetched
			s.cache.set(key, q, ttlFor(inv.Provider))
		}

		if _, err := s.store.SetInvestmentPrice(inv.ID, q.Price, q.At); err != nil {
			results = append(results, SymbolResult{ID: inv.ID, Symbol: inv.Symbol, Error: err.Error()})
			continue
		}
		results = append(results, SymbolResult{ID: inv.ID, Symbol: inv.Symbol, OK: true, Price: q.Price})
	}

	return RefreshResult{
		Investments: s.store.ListInvestments(),
		Results:     results,
		RefreshedAt: time.Now().UTC(),
	}
}

// Search resolves a query to symbols for the given kind ("mf" or "stock").
func (s *Service) Search(ctx context.Context, kind, q string) ([]SearchHit, error) {
	switch kind {
	case "mf":
		return searchMF(ctx, s.shared, q)
	case "stock":
		return s.nse.search(ctx, q)
	case "bse":
		return s.bse.search(ctx, q)
	default:
		return nil, fmt.Errorf("unknown search kind: %q", kind)
	}
}

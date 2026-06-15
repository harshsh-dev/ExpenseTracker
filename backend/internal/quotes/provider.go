// Package quotes fetches live unit prices for investments (mutual funds via
// MFAPI, Indian stocks via NSE with a Yahoo Finance fallback, crypto via
// CoinGecko) so the app can compute current value and profit/loss.
//
// All fetching happens server-side: NSE requires browser-like headers + a
// cookie session, and centralizing it lets us cache and rate-limit across all
// of a user's devices.
package quotes

import (
	"context"
	"time"
)

// Quote is a single normalized price for a symbol.
type Quote struct {
	Price    float64
	Currency string
	At       time.Time
}

// Provider fetches a per-unit price for a symbol it understands.
type Provider interface {
	ID() string
	GetQuote(ctx context.Context, symbol string) (Quote, error)
}

// SearchHit is a symbol-resolution result for autocomplete.
type SearchHit struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// ttlFor returns how long a provider's quote stays fresh in the cache.
func ttlFor(provider string) time.Duration {
	switch provider {
	case "coingecko":
		return 2 * time.Minute
	case "stock", "bse":
		return 15 * time.Minute
	case "mfapi":
		return 6 * time.Hour // NAV is end-of-day
	default:
		return 10 * time.Minute
	}
}

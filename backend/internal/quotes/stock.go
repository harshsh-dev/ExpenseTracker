package quotes

import (
	"context"
	"strings"
)

// stockProvider serves Indian stocks: NSE (with spoofed headers/cookies) as the
// primary source, Yahoo Finance as the fallback when NSE is blocked or empty.
// Symbol is the bare NSE ticker (e.g. "RELIANCE"); Yahoo gets a ".NS" suffix.
type stockProvider struct {
	nse   nseSource
	yahoo yahooSource
}

func (p stockProvider) ID() string { return "stock" }

func (p stockProvider) GetQuote(ctx context.Context, symbol string) (Quote, error) {
	if q, err := p.nse.GetQuote(ctx, symbol); err == nil && q.Price > 0 {
		return q, nil
	}
	ysym := strings.ToUpper(strings.TrimSpace(symbol))
	if !strings.Contains(ysym, ".") {
		ysym += ".NS"
	}
	return p.yahoo.GetQuote(ctx, ysym)
}

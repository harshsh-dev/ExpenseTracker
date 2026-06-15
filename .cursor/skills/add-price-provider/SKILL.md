---
name: add-price-provider
description: Workflow to add an investment price/quote data source to the Money Tracker Go backend for automatic profit/loss — e.g. a new mutual fund, stock, crypto, gold, or forex source. Use when the user wants live prices, current value, or P&L for investments.
---

# Add a Price Provider

Price fetching lives in `backend/internal/quotes/`. Providers are server-side (keys/cookies never reach the client). Read `money-tracker-architecture` first. Existing providers: `mfapi` (MFAPI.in NAV), `stock` (NSE primary → Yahoo fallback), `coingecko`. None need API keys.

## Interface (implemented)

```go
type Quote struct {
    Price    float64
    Currency string
    At       time.Time
}

type Provider interface {
    ID() string                                       // "mfapi", "stock", "coingecko"
    GetQuote(ctx context.Context, symbol string) (Quote, error)
}
```

The `stock` provider is a chain: `stockProvider{ nse, yahoo }` tries NSE (via `nseSession` — cookie bootstrap + spoofed browser headers) and falls back to Yahoo (`symbol + ".NS"`).

## Checklist

```
- [ ] 1. Implement Provider in backend/internal/quotes/<source>.go (return per-unit price in INR + timestamp)
- [ ] 2. Register it in quotes/service.go New() providers map
- [ ] 3. (If new provider id) add it to the frontend providers list + form
- [ ] 4. (Optional) add a search source for autocomplete in Service.Search
- [ ] 5. Set a sensible cache TTL in provider.go ttlFor()
- [ ] 6. Verify: go build/vet, run, POST /api/quotes/refresh
```

## Steps

### 1. Implement
Create `backend/internal/quotes/<source>.go` with a struct holding `*http.Client`, implement `ID()` and `GetQuote`. Return the **per-unit** price (convert to INR if the source differs — currently everything is INR). Use a browser `User-Agent` for sites that need it.

### 2. Register
Add it to the `providers` map in `quotes.New` (`service.go`). `RefreshAll` dispatches by `inv.Provider == ID()`.

### 3. Frontend
If it's a new `provider` id, add an option to the `providers` list in `frontend/src/modules/Investments.tsx` and decide whether the symbol field uses a plain input (like `coingecko`) or `SymbolSearch` (like `mfapi`/`stock`).

### 4. Search (optional)
For autocomplete, add a case in `Service.Search` and wire `GET /api/quotes/search/<kind>`.

### 5. TTL
Add the provider to `ttlFor()` in `provider.go` (crypto short, NAV ~6h, stocks ~15m). Respect free-tier limits; dedupe is automatic via the cache.

### 6. Verify
`cd backend && go build ./... && go vet ./...`; run the server; create an investment with the new provider + `symbol` + `quantity`; `POST /api/quotes/refresh` and confirm `lastPrice`/`lastPriceAt` update and the frontend P/L recomputes. Confirm a snapshot round-trip preserves the cache.

## Guardrails
- Server-side only; never expose keys/cookies to the client or snapshots.
- `GetQuote` returns per-unit price in the entity currency — no totals.
- On failure, return an error (the service keeps the prior cached price) — never zero out data.
- `lastPrice`/`lastPriceAt` remain the only stored derived values.

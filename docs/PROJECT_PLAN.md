# Money Tracker — Build Plan & To-Dos

Roadmap for the **Go API + React/Vite** monorepo. Phases are ordered by dependency. Checked items are already built and verified (backend builds + vets, frontend builds + lints, end-to-end smoke test passed).

Reference: [README.md](../README.md) · [ARCHITECTURE.md](ARCHITECTURE.md). Skills: `money-tracker-architecture`, `add-feature-module`, `add-price-provider`.

**Legend:** 🟢 MVP · 🔵 v1 polish · ⚪ later

---

## Progress overview

```
- [x] Phase 0  — Monorepo + toolchain (Go module, Vite app) 🟢
- [x] Phase 1  — Backend domain (entities, validation, defaults) 🟢
- [x] Phase 2  — In-memory store + JSON snapshot persistence 🟢
- [x] Phase 3  — REST API (chi, generic CRUD, CORS, health) 🟢
- [x] Phase 4  — Backup export / import endpoints 🟢
- [x] Phase 5  — Frontend foundation (client, hooks, layout, UI kit) 🟢
- [x] Phase 6  — Income page 🟢
- [x] Phase 7  — Expenses page (+ categories filter) 🟢
- [x] Phase 8  — Investments page (manual P/L) 🟢
- [x] Phase 9  — Categories page 🟢
- [x] Phase 10 — Dashboard + charts 🔵
- [x] Phase 11 — Backup/restore UI (Settings) 🟢
- [ ] Phase 12 — Tests (Go store/api + frontend) 🔵
- [x] Phase 13 — Live price providers + auto P/L (MFAPI, NSE→Yahoo, CoinGecko) 🔵
- [ ] Phase 14 — Deploy (frontend → Vercel, backend → container host) 🟢
- [ ] Phase 15 — UX polish, PWA, mobile, accessibility 🔵
- [ ] Phase 16 — Backlog / future modules ⚪
```

---

## Done — what exists today

**Backend (`backend/`)**
- `internal/domain` — Income, Expense, Investment, Category structs + `Validate()` + 16 default categories.
- `internal/store` — mutex-guarded in-memory maps; atomic snapshot persistence (temp file + rename); seed-on-empty; `Export`/`Import`.
- `internal/api` — chi router (RequestID, Logger, Recoverer, Timeout, CORS), generic `crud[T]` handlers for all resources, backup export/import, `/health`.
- `cmd/server/main.go` — env config (`PORT`, `DATA_PATH`, `ALLOWED_ORIGINS`), graceful shutdown. `Dockerfile` + `.gitignore`.

**Frontend (`frontend/`)**
- `api/client.ts` + `api/hooks.ts` — typed fetch client + TanStack Query hooks.
- `modules/` — Dashboard (cards + bar/pie charts), Income, Expenses, Investments, Categories, Settings (backup/restore).
- `components/ui.tsx`, `lib/format.ts`, `types.ts`. Tailwind v4, React Router, Vite dev proxy to the API.

---

## Phase 12 — Tests 🔵

```
- [ ] Go: store CRUD + persistence round-trip (load → mutate → reload)
- [ ] Go: snapshot export/import + version rejection
- [ ] Go: api handler tests (httptest) incl. validation 422s
- [ ] Frontend: format helpers + investment P/L derivations (vitest)
- [ ] Frontend: a couple of component/render tests
```

**Acceptance:** `go test ./...` and `npm test` pass.

---

## Phase 13 — Live price providers + auto P/L ✅

```
- [x] backend/internal/quotes: Provider interface { ID, GetQuote } + TTL cache
- [x] CoinGecko provider (crypto, INR, no key)
- [x] MFAPI.in provider (Indian mutual funds, EOD NAV, no key)
- [x] Stock provider: NSE (spoofed headers + cookie session) primary → Yahoo Finance fallback (no key)
- [x] POST /api/quotes/refresh: fetch for investments with symbol+quantity, update lastPrice cache, persist
- [x] GET /api/quotes/search/{mf|stock}: symbol autocomplete
- [x] Dedupe via cache; on failure keep cached price; periodic scheduler (12h, configurable)
- [x] Frontend: "Refresh prices" button + last-updated; provider/symbol/quantity inputs + SymbolSearch autocomplete
```

**Done & verified live:** CoinGecko (BTC), MFAPI (NAV), and the NSE→Yahoo stock chain all returned prices; P/L computes from the cache.

**Possible follow-ups:** XIRR/annualized return (needs `investedOn`/lots), per-investment refresh button, staleness badges.

---

## Phase 14 — Deploy 🟢

```
- [ ] Frontend: set VITE_API_URL; `npm run build`; deploy frontend/ to Vercel
- [ ] Backend: build Docker image; deploy to Render/Railway/Fly.io with a persistent volume at /app/data
- [ ] Set ALLOWED_ORIGINS to the frontend origin
- [ ] Smoke test on a second device; export → import across deployments
```

**Acceptance:** live URLs work cross-device; snapshot export/import round-trips.

---

## Phase 15 — UX polish 🔵

```
- [ ] Date-range / month selector on the dashboard
- [ ] CSV / PDF report export
- [ ] Toasts, optimistic updates, loading skeletons, inline form errors
- [ ] Mobile layout refinements + PWA (installable, offline shell)
- [ ] Accessibility pass; quarterly export reminder
```

---

## Phase 16 — Backlog / future modules ⚪

Each follows `add-feature-module` (Go entity + store + `crud[T]`; TS type + module page).

```
- [ ] Budgets & monthly limits (over-budget alerts)
- [ ] Savings goals
- [ ] Debt / loan tracker (amortization)
- [ ] Recurring bills & income (scheduler)
- [ ] Multi-currency with FX rates
- [ ] Receipt attachments
- [ ] Real database backend (SQLite/Postgres) behind the store package
- [ ] Auth / multi-user
```

---

## Milestones

| Milestone | Phases | Outcome |
|-----------|--------|---------|
| **M1 — Working MVP** | 0–11 | ✅ Track income, expenses, investments, categories; dashboard; backup/restore. |
| **M2 — Hardened + live** | 12–14 | Tests, live prices, deployed. |
| **M3 — Polished v1** | 15 | Reports, PWA, mobile, a11y. |
| **M4 — Beyond** | 16 | Budgets, goals, real DB, auth. |

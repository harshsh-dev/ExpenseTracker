# Architecture

Technical contract for Money Tracker. Read before adding features.

It's a monorepo: a **Go API** (`backend/`) and a **React + Vite SPA** (`frontend/`). The backend is the system of record at runtime (in-memory), durably backed by a JSON snapshot on disk; the frontend is a thin client over the REST API.

---

## 1. Topology

```
React SPA (frontend/)
   │  fetch /api/*  (dev: Vite proxy → :8080)
   ▼
Go API (backend/, chi)            cmd/server
   │                                  │
internal/api  ── generic CRUD ──▶ internal/store (in-memory + mutex)
                                      │ atomic write on every mutation
                                      ▼
                              data/snapshot.json  ←─ export/import
```

**Backend dependency rule:** `cmd → api → store → domain`. `domain` and `store` never import `api`.

---

## 2. Domain entities (`backend/internal/domain`)

Every entity embeds `Base { id, createdAt, updatedAt }` and implements `Validate() error`. The frontend mirrors each as a TypeScript interface in `frontend/src/types.ts`; keep JSON tags and TS fields in sync.

- **Income** (monthly): `source, amount, currency, month (1-12), year, receivedOn, note?`
- **Expense** (daily): `amount, currency, categoryId, subcategory?, date, paymentMethod, note?`
- **Investment**: `name, type, platform?, symbol?, provider, quantity?, amountInvested, currentValue?, currency, investedOn, note?, lastPrice?, lastPriceAt?`
- **Category** (config-driven, user-editable): `name, color, icon?, subcategories[], archived`

Defaults: `domain.DefaultCategories()` seeds the 16 starter categories on first run (only when the store has none).

### Investment derived values (never stored)

```
currentValue = quantity * lastPrice         (else manual currentValue)
profitOrLoss = currentValue - amountInvested
returnPct    = profitOrLoss / amountInvested * 100
```

`lastPrice`/`lastPriceAt` are a **price cache** (refreshed by a future quotes provider), not source-of-truth — the only stored "derived-ish" values.

---

## 3. Store (`backend/internal/store`)

The in-memory database: maps keyed by id, guarded by a `sync.RWMutex`.

- **Persistence:** every mutation calls `persist()`, which writes the full snapshot to a temp file and `rename`s it (atomic). On boot, `load()` rehydrates from disk; if empty, it seeds default categories.
- **Snapshot** (`snapshot.go`): `{ schemaVersion, exportedAt, app, data{ incomes, expenses, investments, categories } }`. This single versioned shape is used for both on-disk persistence and export/import.
- **Per-entity methods:** `List*`, `Create*`, `Update*`, `Delete*` (create assigns id+timestamps; update preserves `createdAt`, bumps `updatedAt`). `Export()` / `Import()` handle whole-snapshot transfer.

**Swapping storage:** everything DB-specific lives here behind plain method signatures. A SQLite/Postgres implementation can replace the maps without touching `api` or `domain`.

---

## 4. API (`backend/internal/api`)

- **Router** (`router.go`): chi with RequestID/RealIP/Logger/Recoverer/Timeout + CORS (origins from `ALLOWED_ORIGINS`). Mounts all four resources and the backup endpoints under `/api`, plus `/health`.
- **Generic CRUD** (`crud.go`): `crud[T]` wires a store's `list/create/update/delete` funcs into REST handlers — one implementation for all resources. On create/update it calls `Validate()` if the payload implements it.
- **Responses** (`respond.go`): JSON helpers; request bodies decoded with `DisallowUnknownFields`.

### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET/POST | `/api/{resource}` | list / create |
| PUT/DELETE | `/api/{resource}/{id}` | update / delete |
| GET | `/api/backup/export` | download full snapshot |
| POST | `/api/backup/import` | validate + replace all data |
| GET | `/health` | liveness |

`resource` ∈ `incomes`, `expenses`, `investments`, `categories`.

Import validation rejects non-`money-tracker` files and snapshots from a newer `schemaVersion`.

---

## 5. Backup, restore & migrations

- **Export:** `GET /api/backup/export` returns the snapshot with a `Content-Disposition` filename; the frontend also offers a client-side download (Settings page).
- **Import:** `POST /api/backup/import` → validate app + version → `store.Import` (replace all + persist).
- **Cross-device:** export on the backend host → import on another deployment; data lives entirely in the snapshot.
- **Migrations:** bump `SchemaVersion` (`snapshot.go`) on breaking shape changes and add a forward migration so older snapshots still import. Forward-only, additive.

---

## 6. Frontend (`frontend/src`)

- **`api/client.ts`** — typed `fetch` wrapper; `resource<T>()` factory yields `list/create/update/remove` per resource; plus `exportSnapshot`/`importSnapshot`. Components never call `fetch` directly.
- **`api/hooks.ts`** — TanStack Query hooks (`useIncomes`, `useIncomeCrud`, …) handling caching + invalidation.
- **`modules/`** — one file per page: `Dashboard`, `Income`, `Expenses`, `Investments`, `Categories`, `Settings`. Each owns its list + form (modal).
- **`components/ui.tsx`** — shared primitives (Card, Button, Field, Input, Select, Modal, Pill, Empty).
- **`lib/format.ts`** — money/date formatting (the only place values are formatted).
- **`types.ts`** — TS mirror of the Go domain + derived helpers (`investmentCurrentValue`, `investmentPnl`).
- **Routing/state:** React Router for pages; React Query for server state; no global client store needed.
- **Dev proxy:** `vite.config.ts` forwards `/api` + `/health` to `http://localhost:8080`.

---

## 7. Price quotes & auto P/L (implemented)

Lives in `backend/internal/quotes/`. All fetching is server-side (NSE needs spoofed browser headers + a cookie session; centralizing also enables caching/rate-limiting). **No API keys required.**

**Providers** (`Provider` interface: `ID()`, `GetQuote(ctx, symbol)`):

| `provider` | Source | Symbol | Notes |
|------------|--------|--------|-------|
| `mfapi` | MFAPI.in (AMFI NAV) | AMFI scheme code (`120503`) | NAV is end-of-day; TTL 6h |
| `stock` | **NSE primary → Yahoo fallback** | NSE ticker (`RELIANCE`) | NSE via `nseSession` (cookie bootstrap + headers); on fail/empty → Yahoo `.NS`; TTL 15m |
| `bse` | **BSE primary → Yahoo fallback** | BSE scrip code (`500180`) | BSE `getScripHeaderData` (browser headers, no cookies); on fail/empty → Yahoo `.BO`; TTL 15m |
| `coingecko` | CoinGecko (`vs_currency=inr`) | coin id (`bitcoin`) | real-time; TTL 2m |
| `manual` | — | — | user enters `currentValue` |

**Service** (`service.go`): `RefreshAll(ctx)` iterates investments that have a non-manual `provider` + `symbol` + `quantity`, dedupes via a TTL `cache`, fetches, and writes the price through `store.SetInvestmentPrice(id, price, at)` (updates the `lastPrice`/`lastPriceAt` cache only, persists snapshot). Returns per-symbol results. A single refresh runs at a time (mutex).

**Endpoints:** `POST /api/quotes/refresh` (refresh all), `GET /api/quotes/search/{mf|stock|bse}?q=` (symbol search). Stock/BSE search runs locally against cached exchange master lists — NSE `EQUITY_L.csv` and BSE `ListofScripData` (both refreshed every 12h) — using ranked matching (exact/prefix/glued-substring/all-tokens/subsequence), since NSE's old autocomplete endpoint was removed. MF search uses MFAPI.

**Scheduler:** `cmd/server` runs a goroutine that refreshes ~10s after boot then every `QUOTES_REFRESH_INTERVAL` (default 12h); disable with `QUOTES_REFRESH=off`. Cadence is deliberately low (NAVs are daily; exchanges must not be hammered).

**Resilience:** per-call context timeout; NSE session re-bootstraps on 401/403; NSE/BSE each fall back to Yahoo (`.NS`/`.BO`); on any fetch failure the previous `lastPrice` is kept so P/L still renders (offline + restored snapshots included). P/L itself is computed in the frontend from the cache.

See the `add-price-provider` skill to add a source.

---

## 8. Conventions

- **Entities mirrored both sides** (Go struct ↔ TS interface); JSON is the contract.
- **Validate server-side** via `Validate()`.
- **No derived data stored** (except the price cache).
- **Money:** numeric + explicit `currency`; format only in the UI.
- **Dates:** ISO strings (`YYYY-MM-DD`).
- **IDs:** random 16-byte hex (server-assigned).
- **`domain`/`store` stay free of HTTP concerns.**

---

## 9. Roadmap-safe extension points

- **New entity/feature** — Go struct + store methods + one `crud[T]` mount; TS type + a module page. See `add-feature-module`.
- **Budgets, savings goals, debt tracker, recurring bills** — new entities, no redesign.
- **Price providers** — implement `QuoteProvider` (`add-price-provider`).
- **Real database** — replace `internal/store` internals; API/domain untouched.
- **Auth / multi-user** — add middleware + an owner field on entities.

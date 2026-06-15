---
name: money-tracker-architecture
description: Architecture, conventions, and domain model for the Money Tracker app — a Go API backend + React/Vite frontend for income, expenses, and investments. Use when working anywhere in this repo (Go handlers/store/domain, React modules/api/types, backup/restore, prices/profit-loss) or when the user mentions income, expenses, investments, categories, snapshot, or deploy.
---

# Money Tracker — Architecture Skill

Monorepo: **`backend/`** (Go, `net/http` + chi) and **`frontend/`** (React + Vite + TS). Data is **in-memory in the Go store, auto-persisted to `data/snapshot.json`**; export/import is the backup + cross-device path. Full detail in `docs/ARCHITECTURE.md`.

## Non-negotiable rules

1. **Backend layering:** `cmd → internal/api → internal/store → internal/domain`. `domain` and `store` never import `api`.
2. **Mirror entities both sides:** a Go struct in `backend/internal/domain` and a matching TS interface in `frontend/src/types.ts`. JSON is the contract — keep tags/fields in sync.
3. **Validate server-side:** entities implement `Validate() error`; the generic `crud[T]` handler calls it. Don't trust the client.
4. **Persistence only via `store`:** handlers call store methods; no file/DB access in `api`. Every mutation persists the snapshot atomically (temp file + rename).
5. **No derived data stored** (totals, returns, savings rate) — compute it. Sole exception: `lastPrice`/`lastPriceAt` (price cache).
6. **Frontend → API only via** `frontend/src/api/client.ts` + `api/hooks.ts`. Components never call `fetch`.
7. **Money & dates:** numeric + explicit `currency`; ISO date strings; format only in `frontend/src/lib/format.ts`.
8. **Forward-compatible backups:** changing the snapshot shape ⇒ bump `SchemaVersion` (`store/snapshot.go`) + add a migration; old snapshots must still import.

## Where things live

| Concern | Location |
|---------|----------|
| Go entities + validation + defaults | `backend/internal/domain/` |
| In-memory store + snapshot persistence | `backend/internal/store/` |
| chi router, generic CRUD, backup | `backend/internal/api/` |
| Server entrypoint | `backend/cmd/server/main.go` |
| FE API client + query hooks | `frontend/src/api/` |
| FE feature pages | `frontend/src/modules/` |
| FE shared UI / formatting / types | `frontend/src/components/ui.tsx`, `lib/format.ts`, `types.ts` |

## Entities

- **Income** (monthly): `source, amount, currency, month, year, receivedOn, note?`
- **Expense** (daily): `amount, currency, categoryId, subcategory?, date, paymentMethod, note?`
- **Investment**: `name, type, platform?, symbol?, provider, quantity?, amountInvested, currentValue?, currency, investedOn, lastPrice?, lastPriceAt?` (current value / P&L computed)
- **Category**: `name, color, icon?, subcategories[], archived`

## API surface

`GET/POST /api/{incomes|expenses|investments|categories}`, `PUT/DELETE /api/{resource}/{id}`, `GET /api/backup/export`, `POST /api/backup/import`, `GET /health`.

## Common changes

- **Add a field:** edit the Go struct + TS interface (+ `Validate()` if required). If persisted shape changes meaningfully, bump `SchemaVersion` + migration.
- **Add an entity/feature:** follow the `add-feature-module` skill.
- **Add a price source:** follow the `add-price-provider` skill.
- **Swap to a real DB:** reimplement `internal/store` internals only; `api`/`domain` untouched.

## Build / verify

```bash
cd backend && go build ./... && go vet ./...
cd frontend && npm run build && npm run lint
```
Dev: backend `go run ./cmd/server` (:8080); frontend `npm run dev` (:5173, proxies `/api`).

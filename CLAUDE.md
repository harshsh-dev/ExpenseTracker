# CLAUDE.md

Guidance for AI agents (and humans) working in this repo. Read this first, then the linked docs as needed.

## What this is

**Money Tracker** — a personal finance app for monthly **income**, daily **expenses** (categorized), and **investments** (with profit/loss). It's a **monorepo**:

- **`backend/`** — Go API (`net/http` + chi). In-memory store, auto-persisted to a JSON snapshot on disk. Export/import endpoints for backup + cross-device transfer.
- **`frontend/`** — React + Vite + TypeScript SPA (Tailwind v4, TanStack Query, React Router, Recharts).

## Source of truth

| Topic | Read |
|-------|------|
| Overview, stack, run, deploy | [README.md](README.md) |
| Architecture & data model | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Build roadmap + to-dos | [docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md) |
| Repo conventions (skill) | `.cursor/skills/money-tracker-architecture/` |
| Adding a feature (skill) | `.cursor/skills/add-feature-module/` |
| Adding a price source (skill) | `.cursor/skills/add-price-provider/` |

## Golden rules

1. **Backend layering:** `cmd/ → internal/api → internal/store → internal/domain`. `domain` and `store` never import `api`.
2. **One entity, mirrored both sides:** a Go struct in `backend/internal/domain` and a matching TS interface in `frontend/src/types.ts`. Keep JSON tags and TS fields in sync.
3. **Validate on the server:** every create/update implements `Validate()` (called by the generic CRUD handler). Never trust the client.
4. **All persistence goes through `store`:** handlers call store methods only; no file/DB access in `api`. Every mutation persists the snapshot atomically.
5. **Never store derived data** (totals, returns, savings rate) — compute it. Sole exception: `lastPrice`/`lastPriceAt` (a quote cache).
6. **Money & dates:** numbers + explicit `currency`; ISO date strings (`YYYY-MM-DD`); format only in the UI (`frontend/src/lib/format.ts`).
7. **Frontend talks to the API only via** `frontend/src/api/client.ts` + the hooks in `api/hooks.ts`. Components never call `fetch` directly.
8. **Backups are forward-compatible:** changing the snapshot shape requires bumping `SchemaVersion` (backend `internal/store/snapshot.go`) and adding a migration; old snapshots must still import.
9. **Secrets** (future price API keys) live only on the backend, never in the frontend bundle or snapshots.

## Layout

```
backend/cmd/server/main.go        # entrypoint
backend/internal/domain/          # entities, validation, default categories
backend/internal/storage/         # blob persistence backends: local file (default) or Firestore
backend/internal/store/           # in-memory store + snapshot persistence (via storage.Blob)
backend/internal/auth/            # Notion OAuth login + sessions (optional; auth.json, not in backups)
backend/internal/notion/          # Notion API client + one-way data sync
backend/internal/api/             # chi router, generic crud, backup/auth/notion handlers
frontend/src/api/                 # client + React Query hooks
frontend/src/modules/             # one folder/file per feature page
frontend/src/components/ui.tsx    # shared UI primitives
frontend/src/types.ts             # TS mirror of the Go domain
```

## Entities

- **Income** (monthly): `source, amount, currency, month, year, receivedOn, note?`.
- **Expense** (daily): `amount, currency, categoryId, subcategory?, date, paymentMethod, note?`.
- **Investment**: `name, type, platform?, symbol?, provider, quantity?, amountInvested, currentValue?, currency, investedOn, lastPrice?, lastPriceAt?`. `currentValue`/P&L are computed when `quantity`+`lastPrice` exist.
- **Category**: `name, color, icon?, subcategories[], archived`.

## Commands

```bash
# backend
cd backend && go run ./cmd/server      # run API on :8080
go build ./... && go vet ./...         # build + vet

# frontend
cd frontend && npm run dev             # dev server on :5173 (proxies /api → :8080)
npm run build                          # tsc + vite build
npm run lint                           # eslint
```

## API surface

`GET/POST /api/{incomes|expenses|investments|categories}`, `PUT/DELETE /api/{resource}/{id}`, `GET /api/backup/export`, `POST /api/backup/import`, `GET /health`.

Auth (active when `NOTION_CLIENT_ID/SECRET` **or** `APP_PASSWORD` set; then all routes except `/health`, `/api/config`, `/api/auth/*` require a session cookie): `GET /api/auth/notion/{login|callback}`, `POST /api/auth/login` (password mode), `GET /api/auth/me`, `POST /api/auth/logout`. Notion sync (token from OAuth login or `NOTION_TOKEN`): `GET /api/notion/status`, `POST /api/notion/sync` (push), `POST /api/notion/pull` (import Notion edits/additions; deletions never propagate).

## Workflow expectations

- Pick up work from [docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md).
- After backend changes: `go build ./... && go vet ./...`. After frontend changes: `npm run build` + `npm run lint`.
- When adding/changing an entity, update **both** the Go struct and the TS type, and the snapshot if persisted.
- Keep the in-memory + snapshot model unless explicitly asked to switch to a real database.

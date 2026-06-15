# Money Tracker

A personal finance tracker for **income**, **daily expenses**, and **investments** — usable across **multiple devices** and designed to be **highly extensible**.

It's a **monorepo**: a **Go** API backend (`backend/`) and a **React + Vite** frontend (`frontend/`). The backend keeps data **in-memory** (fast, simple) and auto-persists every change to a **JSON snapshot on disk**, so a restart/redeploy rehydrates. **Export / import** of that snapshot is the backup and cross-device transfer mechanism — even if the host wipes everything, you re-upload the snapshot and resume exactly where you left off.

---

## 1. What it does

| Module | Cadence | Description |
|--------|---------|-------------|
| **Income** | Monthly | Record salary/other income per month, multiple sources, grouped by month with totals. |
| **Expenses** | Daily | Add expenses with category + subcategory, date, payment method, note; filter by category. |
| **Investments** | Anytime | Track investments (stocks, MF, FD, gold, crypto, …) with **auto-fetched live prices** → current value, **profit/loss** %, and a portfolio summary. |
| **Categories** | Anytime | Fully editable taxonomy (name, color, subcategories); seeded with sensible defaults. |
| **Dashboard** | Live | Income vs expense, net savings + savings rate, portfolio P/L, charts. |
| **Reports** | Weekly / Monthly / Annual | Pick a period, visualize income vs expense, category breakdown, top expenses & income sources, and **download a PDF**. |
| **Backup** | Quarterly / anytime | Download a full JSON snapshot; re-upload on any device to restore. |

> **Investments note:** live prices are fetched **server-side, no API keys** — mutual funds via MFAPI.in (AMFI NAV), Indian stocks via **NSE or BSE (spoofed headers) with a Yahoo Finance fallback** (`.NS`/`.BO`), crypto via CoinGecko (in INR). Use the **Refresh prices** button or the periodic auto-refresh; for assets without a feed (FD, gold, real estate) enter current value manually. Symbol search is built in for funds and both exchanges.

> **Modular by design:** every module above is an independently deployable **feature**. Ship the full app, or trim it to just what you need (see [§7 Feature-wise deployment](#7-feature-wise-deployment)).

---

## 2. Default expense categories

Seeded on first run and **fully editable** from the Categories page. Each has subcategories:

Food & Dining · Groceries · Housing · Utilities · Transportation · Health & Medical · Shopping · Entertainment · Subscriptions · Education · Personal Care · Travel · EMI / Loans · Insurance · Gifts & Donations · Taxes & Fees · Miscellaneous

---

## 3. Tech stack

**Backend (`backend/`)**
- **Go 1.25**, `net/http` + **chi** router, **chi/cors**.
- In-memory store guarded by a mutex; atomic JSON snapshot persistence to disk.
- Generic CRUD handlers; one code path for all four resources.

**Frontend (`frontend/`)**
- **React 19 + TypeScript + Vite**.
- **Tailwind CSS v4** (via `@tailwindcss/vite`).
- **TanStack Query** for server state, **React Router** for routing, **Recharts** for charts, **date-fns**.
- Dev proxy forwards `/api` → the Go backend (no CORS pain in dev).

---

## 4. Project structure

```
money-tracker/
├── backend/                      # Go API
│   ├── cmd/server/main.go        # entrypoint (graceful shutdown)
│   ├── internal/
│   │   ├── domain/               # entities + validation + default categories
│   │   ├── store/                # in-memory store + snapshot persistence
│   │   ├── config/               # FEATURES parsing / feature toggles
│   │   ├── quotes/               # live price providers (MFAPI, NSE, BSE, Yahoo, CoinGecko)
│   │   └── api/                  # chi router, generic CRUD, backup endpoints
│   ├── data/                     # snapshot.json (gitignored runtime data)
│   └── Dockerfile
├── frontend/                     # React + Vite SPA
│   └── src/
│       ├── api/                  # fetch client + React Query hooks
│       ├── components/           # shared UI primitives
│       ├── lib/                  # formatting, report aggregation, PDF generation
│       ├── features.tsx          # feature-flag context (reads /api/config)
│       ├── modules/              # Dashboard, Income, Expenses, Investments, Reports, Categories, Settings
│       └── types.ts              # types mirroring the Go domain
├── docs/                         # ARCHITECTURE.md, PROJECT_PLAN.md
├── CLAUDE.md
└── .cursor/skills/               # agent skills
```

---

## 5. Getting started (local)

Two terminals:

```bash
# Terminal 1 — backend (http://localhost:8080)
cd backend
go run ./cmd/server

# Terminal 2 — frontend (http://localhost:5173)
cd frontend
npm install        # first time only
npm run dev
```

Open http://localhost:5173. The Vite dev server proxies API calls to the backend automatically.

### Backend environment variables

| Var | Default | Purpose |
|-----|---------|---------|
| `PORT` | `8080` | Listen port |
| `DATA_PATH` | `data/snapshot.json` | Snapshot file location |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated CORS origins (set to your frontend URL in prod) |
| `FEATURES` | `all` | Which features to enable, comma-separated (`income,expenses,investments,categories,dashboard,report,backup`). `all`/empty = full app. See [§7](#7-feature-wise-deployment). |
| `QUOTES_REFRESH` | `on` | Set `off` to disable the periodic price auto-refresh |
| `QUOTES_REFRESH_INTERVAL` | `12h` | How often prices auto-refresh (Go duration) |

---

## 6. Deploy

The frontend can go on **Vercel** (static build). The backend is **stateful** (in-memory + disk snapshot), so it needs a persistent host, not Vercel serverless.

```bash
# frontend
cd frontend && npm run build      # outputs dist/  (set VITE_API_URL to the backend URL)

# backend (container)
cd backend && docker build -t money-tracker-api .
docker run -p 8080:8080 -v $PWD/data:/app/data money-tracker-api
```

- **Frontend:** deploy `frontend/` to Vercel; set `VITE_API_URL` to the backend's public URL (optionally `VITE_FEATURES` to trim modules at build time).
- **Backend:** deploy the Docker image to Render / Railway / Fly.io / a VPS, with a **persistent volume mounted at `/app/data`** so the snapshot survives restarts. Set `ALLOWED_ORIGINS` to the frontend origin and `FEATURES` to the modules you want live.
- The snapshot is your safety net: download it quarterly so you can restore anywhere.

---

## 7. Feature-wise deployment

The app is **modular**: each capability is a toggleable **feature**, so the same codebase can ship full-fledged or trimmed to a subset. The backend's `FEATURES` env var is the single source of truth — it decides which **API routes** are mounted and advertises the resolved set at **`GET /api/config`**, which the frontend reads to show only the relevant **navigation, pages, and dashboard cards**.

**Features:** `dashboard`, `income`, `expenses`, `investments`, `categories`, `report`, `backup`.

```bash
# Full app (default)
FEATURES=all go run ./cmd/server

# Lean expense tracker (categories auto-enabled as a dependency of expenses)
FEATURES=income,expenses go run ./cmd/server

# Portfolio-only deployment
FEATURES=investments,dashboard go run ./cmd/server
```

Notes:
- **Dependencies are auto-resolved:** enabling `expenses` also enables `categories` (expenses are categorized). Unknown names are ignored; an empty/invalid list safely falls back to `all`.
- **Disabled routes return 404** and their data is never fetched by the frontend — no dead nav links, no empty pages.
- **Investments off** also skips the price-refresh scheduler and `/api/quotes/*`.
- **Optional frontend-only override:** set `VITE_FEATURES` at build time (same syntax) to trim a static frontend independently of the backend. When unset, `/api/config` drives everything.

---

## 8. Extensibility

- **Generic CRUD** on both sides: adding an entity is mostly schema + one registration line.
- **Self-contained frontend modules** under `src/modules/`.
- **Config-driven categories** (data, not code).
- **Versioned snapshots** (`schemaVersion` + migrations) so backups keep importing as the model evolves.
- **Storage is swappable**: the store is isolated behind one package; a SQLite/Postgres backend can replace in-memory later without touching handlers.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full design and [docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md) for the roadmap.

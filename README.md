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
| **Backup** | Quarterly / anytime | Download a full JSON snapshot; re-upload on any device to restore. |

> **Investments note:** live prices are fetched **server-side, no API keys** — mutual funds via MFAPI.in (AMFI NAV), Indian stocks via **NSE (spoofed headers) with a Yahoo Finance fallback**, crypto via CoinGecko (in INR). Use the **Refresh prices** button or the periodic auto-refresh; for assets without a feed (FD, gold, real estate) enter current value manually. Symbol autocomplete is built in for funds and stocks.

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
│   │   └── api/                  # chi router, generic CRUD, backup endpoints
│   ├── data/                     # snapshot.json (gitignored runtime data)
│   └── Dockerfile
├── frontend/                     # React + Vite SPA
│   └── src/
│       ├── api/                  # fetch client + React Query hooks
│       ├── components/           # shared UI primitives
│       ├── lib/                  # formatting helpers
│       ├── modules/              # Dashboard, Income, Expenses, Investments, Categories, Settings
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

- **Frontend:** deploy `frontend/` to Vercel; set `VITE_API_URL` to the backend's public URL.
- **Backend:** deploy the Docker image to Render / Railway / Fly.io / a VPS, with a **persistent volume mounted at `/app/data`** so the snapshot survives restarts. Set `ALLOWED_ORIGINS` to the frontend origin.
- The snapshot is your safety net: download it quarterly so you can restore anywhere.

---

## 7. Extensibility

- **Generic CRUD** on both sides: adding an entity is mostly schema + one registration line.
- **Self-contained frontend modules** under `src/modules/`.
- **Config-driven categories** (data, not code).
- **Versioned snapshots** (`schemaVersion` + migrations) so backups keep importing as the model evolves.
- **Storage is swappable**: the store is isolated behind one package; a SQLite/Postgres backend can replace in-memory later without touching handlers.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full design and [docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md) for the roadmap.

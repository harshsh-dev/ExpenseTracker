# Money Tracker — YouTube Video Script

**Suggested title:** *Build Your Own Finance App — Open Source, Self-Hostable, Feature-Modular*

**Suggested length:** ~8–10 minutes  
**Repo:** https://github.com/harshsh-dev/ExpenseTracker  
**Tone:** Clear, enthusiastic, developer-friendly — not overly salesy.

---

## [0:00 – 0:30] HOOK

**[On screen: Dashboard with charts, dark mode toggle, investment P/L updating]**

> Most finance apps lock your data in someone else's cloud, charge subscriptions, and give you features you never asked for.
>
> What if you could run your **own** expense tracker — on your laptop, a Raspberry Pi, or your cloud — track income, daily expenses, and investments with **live Indian stock prices**, and only enable the features you actually need?
>
> That's **Money Tracker** — open source, self-hostable, and built to grow with you.

**[On screen text: `github.com/harshsh-dev/ExpenseTracker`]**

---

## [0:30 – 1:15] WHAT IS IT?

> Money Tracker is a personal finance app I built as a **monorepo**: a **Go backend** for the API and a **React + Vite frontend** for the UI.
>
> It's designed for real life:
> - Log **monthly income** — salary, freelance, multiple sources.
> - Track **daily expenses** with categories and subcategories.
> - Monitor **investments** — mutual funds, Indian stocks on NSE and BSE, crypto — with **automatic profit and loss**.
> - See everything on a **dashboard**, generate **weekly, monthly, or annual reports**, and **download them as PDF**.
>
> Your data stays on **your** server. Every change is saved to a JSON snapshot on disk. And if you ever redeploy or switch devices, you **export and import** that snapshot — you're back exactly where you left off.

**[B-roll: Income page → Expenses with category pills → Investments with Refresh prices → Reports PDF download]**

---

## [1:15 – 4:00] FEATURE WALKTHROUGH (feature-wise)

### Dashboard
> The **Dashboard** gives you the big picture: total income, total expenses, net savings, savings rate, and portfolio profit/loss. Bar charts show income vs expense over the last six months. The **income allocation pie** shows how much of your income went to each expense category — and what's still unspent.

### Income
> **Income** is monthly. Add salary or other sources, pick the month and year, and optionally the date you received it. Everything groups by month with running totals.

### Expenses
> **Expenses** are daily. Pick a category — Food, Transport, Shopping, and sixteen defaults out of the box — add subcategories, payment method, date, and notes. Filter and review by category anytime.

### Categories
> **Categories** are fully editable. Rename them, change colors, add subcategories, archive old ones. They're data, not hard-coded — so the app adapts to how *you* spend.

### Investments
> **Investments** is where it gets interesting. Add mutual funds, NSE stocks, BSE stocks, or crypto. The backend fetches live prices — **no API keys needed**:
> - Mutual funds via MFAPI — AMFI NAV.
> - Indian stocks: **NSE first**, Yahoo Finance as fallback.
> - BSE stocks: BSE API with Yahoo fallback.
> - Crypto via CoinGecko in INR.
>
> Symbol search is built in — type "HDFC Bank" or "Reliance" and pick the ticker. Hit **Refresh prices** and see current value and P/L instantly. For FDs, gold, or real estate, enter the value manually.

### Reports
> **Reports** let you pick **weekly, monthly, or annual** periods, navigate back and forth in time, and visualize:
> - Income vs expense trends
> - Income allocation by category
> - Top expenses and income sources
>
> Then **download a PDF** with charts, summary tables, and page numbers — ready to share or archive.

### Backup
> **Backup** exports your entire snapshot as JSON. Import it on another machine or after a redeploy. Think of it as your quarterly safety net and cross-device sync.

**[On screen text: 7 modules — Dashboard · Income · Expenses · Categories · Investments · Reports · Backup]**

---

## [4:00 – 6:00] TECHNICAL DEEP DIVE — HOW EXTENSIBLE IS IT?

> Now the part developers care about: **how extensible is this, technically?**
>
> Very. That was a core design goal — not a afterthought.

### Clean architecture
> The backend follows strict layering:
> ```
> cmd → api → store → domain
> ```
> HTTP handlers never touch the filesystem directly. Domain models validate themselves. The store owns all persistence. You can swap the in-memory store for SQLite or Postgres later **without rewriting the API**.

### Mirror entities both sides
> Every entity is a Go struct **and** a matching TypeScript interface. JSON is the contract. Add a field on one side, add it on the other — done.

### Generic CRUD
> One generic CRUD handler powers incomes, expenses, investments, and categories. Adding a new entity — say **budgets** or **recurring bills** — is mostly:
> 1. Define the struct + `Validate()`
> 2. Add store methods
> 3. Register one line in the router
> 4. Add a React module page
>
> There's a step-by-step skill in the repo for exactly this.

### Pluggable price providers
> Investments use a **Provider interface**: `ID()` and `GetQuote()`. MFAPI, NSE, BSE, Yahoo, CoinGecko are all separate providers. Want gold prices from RBI or US stocks? Implement the interface, register it — the refresh service picks it up automatically.

### Feature flags — deploy feature-wise
> This is unique. Set one environment variable:
> ```
> FEATURES=all
> ```
> for the full app, or:
> ```
> FEATURES=income,expenses
> ```
> for a lean expense tracker, or:
> ```
> FEATURES=investments,dashboard
> ```
> for a portfolio-only deployment.
>
> The backend mounts **only the routes you enable**. The frontend reads `/api/config` and hides nav, pages, and API calls for disabled features. Same codebase — different products.

### Versioned backups
> Snapshots have a `schemaVersion`. When the data model evolves, bump the version and add a migration. Old backups still import. Forward-compatible by design.

### Frontend modularity
> Each page is one file under `modules/`. Shared UI primitives, TanStack Query hooks, a single API client — components never call `fetch` directly. Dark mode, responsive layout, Recharts for visualization.

**[On screen: ARCHITECTURE.md diagram — React → Go API → store → snapshot.json]**

> In short: new **entity**? Follow the feature-module skill. New **price source**? Follow the price-provider skill. New **deployment shape**? Toggle `FEATURES`. The bones are already there.

---

## [6:00 – 7:30] HOST IT YOUR WAY — CLOUD OR PERSONAL

> You own the deployment. No vendor lock-in.

### On your personal device
> Run locally in two terminals — Go backend on port 8080, Vite frontend on 5173. Or build the frontend and serve it behind nginx. Run on a **home server**, **NAS**, or **Raspberry Pi** with Docker:
> ```bash
> docker build -t money-tracker-api .
> docker run -p 8080:8080 -v ./data:/app/data money-tracker-api
> ```
> Mount a volume at `/app/data` so your snapshot survives restarts.

### On the cloud
> **Frontend** — static build, deploy to **Vercel**, Netlify, or Cloudflare Pages. Set `VITE_API_URL` to your backend URL.
>
> **Backend** — needs a **persistent disk** (it's stateful, not serverless). Deploy the Docker image to **Railway**, **Render**, **Fly.io**, or any VPS. Mount persistent storage, set `ALLOWED_ORIGINS`, and you're live.
>
> Same app, same data model — whether it's on your desk or in Mumbai.

**[On screen text: Personal ✓ · Cloud ✓ · Docker ✓ · No API keys for prices ✓]**

---

## [7:30 – 8:30] OPEN SOURCE — CONTRIBUTE & EXTEND

> Money Tracker is **open source** under the MIT license.
>
> Link is in the description: **github.com/harshsh-dev/ExpenseTracker**
>
> If this project helps you, here's how you can make it better:
> - **Star the repo** so others find it.
> - **Open an issue** for bugs or feature ideas — budgets, recurring bills, multi-user auth, SMS import, whatever you need.
> - **Send a pull request** — the architecture doc, Cursor skills, and project plan are in the repo so contributors know exactly where to plug in.
>
> Some ideas on the roadmap: automated tests, PWA for mobile, XIRR for investments, deployment guides. Pick one and ship it.
>
> This isn't a finished product behind a paywall. It's a **foundation** — host it yourself, trim it to what you need, and help the community add the next feature.

---

## [8:30 – 9:00] OUTRO

> So — **Money Tracker**: income, expenses, investments with live NSE and BSE prices, reports with PDF export, modular feature deployment, self-hostable on cloud or personal hardware, and fully open source.
>
> Clone it, run it, break it, improve it. Link below.
>
> If you want a follow-up — deployment walkthrough on Railway, adding a new feature live, or NSE price fetching deep dive — drop a comment.
>
> Thanks for watching. See you in the repo.

**[End screen: GitHub repo URL + Subscribe]**

---

## Production notes (for you, not on camera)

| Section | B-roll / screen recording |
|---------|---------------------------|
| Hook | Dashboard stats animating, dark mode toggle |
| Features | Quick cuts: each module 10–15 sec |
| Tech | ARCHITECTURE.md diagram, folder tree in IDE, `FEATURES=investments` demo |
| Hosting | Docker run terminal, Render/Railway dashboard (optional) |
| CTA | GitHub repo page, Issues tab, README feature table |

**Keywords for description:** open source expense tracker, self hosted finance app, Go React monorepo, Indian stock tracker NSE BSE, mutual fund NAV tracker, personal finance self host, feature flags deployment, Docker finance app

**GitHub link (description):** https://github.com/harshsh-dev/ExpenseTracker

# Firebase Launch Plan — Money Tracker

Step-by-step plan to launch Money Tracker on **Google Firebase / Google Cloud**, with honest trade-offs for the **stateful Go backend**.

---

## 1. Executive summary

| Layer | Firebase / GCP service | Fits today? |
|-------|--------------------------|-------------|
| **Frontend** (React SPA) | **Firebase Hosting** | ✅ Yes — drop-in |
| **Backend** (Go API) | **Cloud Run** (container) | ⚠️ Yes, but disk is ephemeral |
| **Data persistence** | **Cloud Storage** bucket *or* **Compute Engine** persistent disk | ⚠️ Needs choice (see §3) |
| **Auth** (optional later) | Firebase Authentication | ⚪ Not implemented yet |
| **Database** (optional later) | Firestore | ⚪ Would replace in-memory store (big refactor) |

**Recommended launch path (fastest, minimal code change):**

```
Users → Firebase Hosting (static React)
          ↓  VITE_API_URL
        Cloud Run (Go Docker image)
          ↓  snapshot read/write
        Cloud Storage bucket (small adapter)  OR  Compute Engine VM + persistent disk (zero code change)
```

Firebase does **not** run long-lived Go servers natively (Cloud Functions are short-lived/serverless). The backend belongs on **Cloud Run** or **Compute Engine** in the **same GCP project** linked to your Firebase app.

---

## 2. Architecture options

### Option A — Firebase Hosting + Cloud Run + Cloud Storage *(recommended production)*

```
┌─────────────────────────────────────────────────────────┐
│  Firebase project (GCP)                                  │
│                                                          │
│  ┌──────────────────┐      HTTPS       ┌──────────────┐ │
│  │ Firebase Hosting │ ───────────────► │  Cloud Run   │ │
│  │  (frontend/dist) │   /api/* proxy   │  Go API      │ │
│  └──────────────────┘   or direct URL  └──────┬───────┘ │
│                                                 │         │
│                                        ┌────────▼───────┐ │
│                                        │ Cloud Storage  │ │
│                                        │ snapshot.json  │ │
│                                        └────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

- **Pros:** Serverless scaling, pay-per-use, fits “cloud launch”
- **Cons:** Requires a **small backend change** so `persist()` / `load()` use GCS instead of local file (Cloud Run filesystem is ephemeral)

### Option B — Firebase Hosting + Compute Engine *(fastest, no backend code change)*

```
Firebase Hosting → VM (e2-small) → Docker + volume mount → existing snapshot.json
```

- **Pros:** Run **existing Dockerfile as-is**; persistent disk survives reboots
- **Cons:** Always-on VM cost (~$5–15/mo); you manage updates/SSH

### Option C — Firebase Hosting only + backend elsewhere

Host frontend on Firebase; keep backend on Railway/Render/Fly.io (already documented in README).

- **Pros:** Zero GCP backend work
- **Cons:** Not “full Firebase”; two vendors

**Pick A for a proper GCP launch. Pick B if you want live this weekend without touching Go code.**

---

## 3. Pre-launch checklist

### Accounts & tools

- [ ] Google account + [Firebase Console](https://console.firebase.google.com)
- [ ] Enable **billing** on the linked GCP project (Blaze plan — required for Cloud Run custom domains; Hosting free tier still applies)
- [ ] Install locally:
  ```bash
  npm install -g firebase-tools
  gcloud auth login
  gcloud auth application-default login
  ```
- [ ] Node 20+ and Go 1.25+ (already in repo)

### Domain (optional)

- [ ] Custom domain for Hosting (e.g. `app.yourdomain.com`)
- [ ] Subdomain for API (e.g. `api.yourdomain.com` → Cloud Run)

### Secrets & config (never commit)

| Variable | Where | Example |
|----------|-------|---------|
| `VITE_API_URL` | Firebase Hosting build | `https://api-xxxxx.run.app` |
| `ALLOWED_ORIGINS` | Cloud Run env | `https://your-app.web.app,https://app.yourdomain.com` |
| `FEATURES` | Cloud Run env | `all` or `income,expenses,investments` |
| `DATA_PATH` or `GCS_BUCKET` | Cloud Run env | see Option A/B |
| `QUOTES_REFRESH` | Cloud Run env | `on` |
| `QUOTES_REFRESH_INTERVAL` | Cloud Run env | `12h` |

---

## 4. Phase-by-phase launch (Option A — recommended)

### Phase 0 — Create Firebase / GCP project (30 min)

1. Firebase Console → **Add project** → name e.g. `money-tracker-prod`
2. Link to GCP; note **Project ID** (e.g. `money-tracker-prod-abc123`)
3. Enable APIs (GCP Console → APIs & Services):
   - Cloud Run API
   - Artifact Registry API
   - Cloud Storage API
   - Cloud Build API (optional, for CI)
4. Firebase Console → **Hosting** → Get started (creates default site)

```bash
firebase login
firebase use --add   # select your project
```

---

### Phase 1 — Frontend on Firebase Hosting (1–2 hours)

1. **Build** with production API URL:
   ```bash
   cd frontend
   npm ci
   VITE_API_URL=https://YOUR_CLOUD_RUN_URL npm run build
   ```
   Output: `frontend/dist/`

2. **Init Firebase** at repo root (one time):
   ```bash
   firebase init hosting
   ```
   - Public directory: `frontend/dist`
   - Single-page app: **Yes** (rewrite all routes to `index.html`)
   - Do **not** overwrite existing files if prompted

3. **`firebase.json`** (example):
   ```json
   {
     "hosting": {
       "public": "frontend/dist",
       "ignore": ["firebase.json", "**/.*", "**/node_modules/**"],
       "rewrites": [
         { "source": "**", "destination": "/index.html" }
       ],
       "headers": [
         {
           "source": "**/*.@(js|css)",
           "headers": [{ "key": "Cache-Control", "value": "max-age=31536000" }]
         }
       ]
     }
   }
   ```

4. **Deploy frontend:**
   ```bash
   cd frontend && npm run build   # with VITE_API_URL set
   cd .. && firebase deploy --only hosting
   ```
   Live at: `https://<project-id>.web.app`

5. **Optional:** Firebase Hosting → Custom domain → add DNS records

**Note:** Until Cloud Run is up, set `VITE_API_URL` to a placeholder; rebuild and redeploy after Phase 2.

---

### Phase 2 — Backend on Cloud Run (2–3 hours)

1. **Artifact Registry** (one time):
   ```bash
   gcloud artifacts repositories create money-tracker \
     --repository-format=docker \
     --location=asia-south1   # Mumbai — low latency for India
   ```

2. **Build & push** (from repo root):
   ```bash
   gcloud builds submit backend \
     --tag asia-south1-docker.pkg.dev/PROJECT_ID/money-tracker/api:latest
   ```

3. **Deploy Cloud Run:**
   ```bash
   gcloud run deploy money-tracker-api \
     --image asia-south1-docker.pkg.dev/PROJECT_ID/money-tracker/api:latest \
     --region asia-south1 \
     --platform managed \
     --allow-unauthenticated \
     --port 8080 \
     --memory 512Mi \
     --cpu 1 \
     --min-instances 0 \
     --max-instances 3 \
     --set-env-vars "FEATURES=all,QUOTES_REFRESH=on,ALLOWED_ORIGINS=https://PROJECT_ID.web.app"
   ```
   Copy the service URL (e.g. `https://money-tracker-api-xxxxx.asia-south1.run.app`).

4. **Rebuild frontend** with `VITE_API_URL=<Cloud Run URL>` and redeploy Hosting.

5. **Smoke test:**
   ```bash
   curl https://YOUR_RUN_URL/health
   curl https://YOUR_RUN_URL/api/config
   ```

---

### Phase 3 — Persistent data on Cloud Storage *(required for Cloud Run)*

Cloud Run containers **lose local disk** on restart. Today the app writes `data/snapshot.json` on every mutation.

**Task:** Add optional GCS-backed persistence (estimated **4–8 hours** dev):

| Step | Work |
|------|------|
| 3.1 | Create bucket `gs://PROJECT_ID-money-tracker-data` (uniform access, no public read) |
| 3.2 | Grant Cloud Run service account `roles/storage.objectAdmin` on bucket |
| 3.3 | Env: `STORAGE_BACKEND=gcs`, `GCS_BUCKET=PROJECT_ID-money-tracker-data`, `GCS_OBJECT=snapshot.json` |
| 3.4 | In `internal/store`: if `GCS_BUCKET` set → `load()` download object, `persist()` upload (atomic: temp object + compose/rename pattern) |
| 3.5 | Fallback to `DATA_PATH` file when GCS unset (local dev unchanged) |
| 3.6 | Test: create data → force new Cloud Run revision → data still there |

**Until Phase 3 is done:** use **Option B (Compute Engine)** for production, or accept data loss on Cloud Run cold restarts (not acceptable for prod).

**Interim workaround:** Rely on user **Backup export** only — document that Cloud Run preview is ephemeral. Not recommended for real launch.

---

### Phase 4 — CORS, security, custom API domain (1 hour)

1. **CORS:** Set `ALLOWED_ORIGINS` to exact Hosting URLs (no `*` in prod):
   ```
   https://PROJECT_ID.web.app,https://PROJECT_ID.firebaseapp.com,https://app.yourdomain.com
   ```

2. **Cloud Run custom domain** (optional):
   - Cloud Run → Manage custom domains → map `api.yourdomain.com`
   - Update `VITE_API_URL` and redeploy frontend

3. **Firebase App Check** (optional later): reduce API abuse on public Cloud Run URL

4. **Rate limiting** (optional): Cloud Armor or API Gateway in front of Cloud Run

---

### Phase 5 — CI/CD with GitHub Actions (2–3 hours)

**Workflow outline:**

```yaml
# .github/workflows/deploy-firebase.yml
on:
  push:
    branches: [main]

jobs:
  deploy-backend:
    # gcloud builds submit + gcloud run deploy
    # needs: GCP_SA_KEY secret, PROJECT_ID

  deploy-frontend:
    needs: deploy-backend
    # npm ci && VITE_API_URL=${{ secrets.API_URL }} npm run build
    # firebase deploy --only hosting --token ${{ secrets.FIREBASE_TOKEN }}
```

Secrets to add in GitHub:
- `GCP_SA_KEY` — service account JSON (Cloud Run Admin, Storage Admin, Cloud Build)
- `FIREBASE_TOKEN` — `firebase login:ci`
- `VITE_API_URL` — Cloud Run URL

---

### Phase 6 — Monitoring & backup discipline (ongoing)

| Item | Tool |
|------|------|
| Uptime | Cloud Monitoring alert on `/health` |
| Logs | Cloud Run logs + optional Firebase Analytics (frontend only) |
| Errors | Cloud Error Reporting |
| User backup | Remind users: Settings → Download backup quarterly |
| Ops backup | GCS bucket **versioning** + lifecycle (keep 90 days of snapshot versions) |
| Cost alert | GCP billing budget alert at ₹500 / $10 |

---

## 5. Option B quick path — Compute Engine (no Go changes)

If you want Firebase Hosting **today** without Phase 3:

1. Create **e2-small** VM in `asia-south1`, Ubuntu 22.04
2. Attach **persistent disk** 10 GB mounted at `/data`
3. Install Docker; run:
   ```bash
   docker run -d --restart unless-stopped \
     -p 8080:8080 \
     -v /data:/app/data \
     -e ALLOWED_ORIGINS=https://PROJECT_ID.web.app \
     -e FEATURES=all \
     asia-south1-docker.pkg.dev/PROJECT_ID/money-tracker/api:latest
   ```
4. Put **HTTPS** in front: Caddy/nginx on VM, or Google Cloud Load Balancer
5. Point `VITE_API_URL` to VM HTTPS URL; deploy Hosting

**Cost:** ~$12–18/mo (VM + disk + egress). Simpler ops, same Dockerfile.

---

## 6. What NOT to use Firebase for (with current codebase)

| Service | Why not (today) |
|---------|------------------|
| **Cloud Functions** | Go API is a long-running chi server with in-memory store + scheduler; not a function-per-route model |
| **Firestore only** | Would replace entire `internal/store`; weeks of work |
| **Firebase Realtime DB** | Same — full rewrite |
| **Hosting for API** | Hosting serves static files only; API must be Cloud Run / VM |

---

## 7. Environment matrix

### Local dev (unchanged)

```bash
cd backend && go run ./cmd/server
cd frontend && npm run dev   # proxies /api → :8080
```

### Firebase prod

| Component | Config |
|-----------|--------|
| Hosting | `firebase deploy --only hosting` |
| API | Cloud Run URL |
| Frontend build | `VITE_API_URL=https://api.example.com` |
| Backend | `ALLOWED_ORIGINS=https://app.example.com` |
| Features | `FEATURES=all` or subset |
| Storage | `GCS_BUCKET=...` (Option A) or VM volume (Option B) |

---

## 8. Launch timeline (estimate)

| Week | Milestone |
|------|-----------|
| **Week 1** | Phase 0–2: Firebase project, Hosting live, Cloud Run deployed, manual smoke test |
| **Week 1–2** | Phase 3: GCS persistence **OR** Option B VM |
| **Week 2** | Phase 4: Custom domains, CORS hardened |
| **Week 2–3** | Phase 5: GitHub Actions deploy on push to `main` |
| **Week 3** | Phase 6: Monitoring, README “Deploy to Firebase” section, YouTube / docs update |

---

## 9. Post-launch README snippet (for users)

Add to README after launch:

```markdown
### Deploy to Firebase (Google Cloud)

- **Frontend:** Firebase Hosting — see [docs/FIREBASE_LAUNCH_PLAN.md](docs/FIREBASE_LAUNCH_PLAN.md)
- **Backend:** Cloud Run (Docker) + Cloud Storage for snapshots
- **Quick alternative:** Firebase Hosting + Compute Engine VM (no backend code changes)
```

---

## 10. Decision record

| Question | Recommendation |
|----------|----------------|
| Host frontend on Firebase? | **Yes** |
| Host backend on Firebase Functions? | **No** — use Cloud Run |
| Persistence on Cloud Run? | **Cloud Storage adapter** (Phase 3) or **VM disk** (Option B) |
| Region for India users? | `asia-south1` (Mumbai) |
| Blaze plan required? | **Yes** for Cloud Run + custom domain |
| Open source repo | Unchanged — [github.com/harshsh-dev/ExpenseTracker](https://github.com/harshsh-dev/ExpenseTracker) |

---

## 11. Immediate next actions (your todo)

1. [ ] Create Firebase project + enable billing
2. [ ] Choose **Option A** (Cloud Run + GCS) vs **Option B** (VM, zero code change)
3. [ ] `firebase init hosting` + first deploy of static frontend
4. [ ] Build & deploy Docker image to Cloud Run (or VM)
5. [ ] Set `VITE_API_URL` + `ALLOWED_ORIGINS`; end-to-end test Income → Expense → Backup
6. [ ] If Option A: implement GCS snapshot backend (track as GitHub issue)
7. [ ] Add GitHub Actions deploy workflow
8. [ ] Update README + announce

---

*Related: [README.md](../README.md) · [ARCHITECTURE.md](ARCHITECTURE.md) · [YOUTUBE_SCRIPT.md](YOUTUBE_SCRIPT.md)*

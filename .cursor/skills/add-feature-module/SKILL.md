---
name: add-feature-module
description: Step-by-step workflow to add a new entity/feature (e.g. budgets, savings goals, debt tracker, recurring bills) across the Go backend and React/Vite frontend of Money Tracker, keeping it extensible. Use when the user asks to add a new tracked entity, section, page, or feature.
---

# Add a Feature Module

Adds a new resource end-to-end: a Go entity + store methods + REST mount, and a TS type + frontend page. Read `money-tracker-architecture` first. A feature mirrors the existing four resources, so copy their patterns.

## Checklist

```
- [ ] 1. Go entity struct + Validate() (backend/internal/domain)
- [ ] 2. Add to snapshot if persisted (store/snapshot.go) + bump SchemaVersion + migration if shape changes
- [ ] 3. Store: map field + List/Create/Update/Delete + load/replace wiring (backend/internal/store)
- [ ] 4. Mount crud[T] for the resource in the router (backend/internal/api/router.go)
- [ ] 5. TS interface mirroring the Go struct (frontend/src/types.ts)
- [ ] 6. API resource + hooks (frontend/src/api/client.ts, api/hooks.ts)
- [ ] 7. Frontend page (frontend/src/modules/<Feature>.tsx) + route + nav (App.tsx)
- [ ] 8. Verify: go build/vet, npm build/lint, manual CRUD + snapshot round-trip
```

## Steps

### 1. Go entity
In `backend/internal/domain`, embed `Base`, add JSON tags, implement `Validate() error` (mirror `validate.go`).

### 2. Snapshot
If the entity is persisted, add its slice to `SnapshotData` in `store/snapshot.go`. Bump `SchemaVersion` and add a forward migration if you change an existing shape; new additive slices are backward-safe.

### 3. Store
In `backend/internal/store`: add a `map[string]domain.X`, init it in `New`, include it in `replaceLocked`/`snapshotLocked` + a `sorted*` helper, and add `ListX/CreateX/UpdateX/DeleteX` (copy the income methods in `crud.go`). Create assigns `newBase()`; update preserves `CreatedAt` and bumps `UpdatedAt`; every method ends with `s.persist()`.

### 4. Router
In `api/router.go`, add one `crud[domain.X]{ list:s.ListX, create:s.CreateX, update:s.UpdateX, delete:s.DeleteX }.mount(r, "/xs")`.

### 5. TS type
Add an interface extending `Base` in `frontend/src/types.ts`, fields matching the Go JSON tags exactly.

### 6. API + hooks
Add `xs: resource<X>('/api/xs')` in `client.ts`; add `useXs`/`useXCrud` in `api/hooks.ts` (copy existing).

### 7. Page
Create `frontend/src/modules/<Feature>.tsx` (list + modal form, copy Income/Expenses). Add a `<Route>` in `App.tsx` and an entry in its `nav` array.

### 8. Verify
`cd backend && go build ./... && go vet ./...`; `cd frontend && npm run build && npm run lint`. Manually create/edit/delete the entity; export then import a snapshot and confirm the new data round-trips.

## Guardrails
- `domain`/`store` never import `api`; components never call `fetch` directly.
- Validate server-side; never store derived values.
- Keep the Go struct and TS interface in sync.
- Backups stay forward-compatible.

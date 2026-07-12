package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"moneytracker/backend/internal/auth"
	"moneytracker/backend/internal/config"
	"moneytracker/backend/internal/domain"
	"moneytracker/backend/internal/notion"
	"moneytracker/backend/internal/quotes"
	"moneytracker/backend/internal/store"
)

// NewRouter builds the HTTP handler for the API. Only routes for enabled
// features are mounted; the resolved feature set is advertised at /api/config.
// When Notion login is configured (a.Enabled), every route except /health,
// /api/config and /api/auth/* requires a session.
func NewRouter(s *store.Store, q *quotes.Service, feats config.Features, allowedOrigins []string, a *auth.Service, y *notion.Syncer) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type", "Authorization"},
		// Always allow credentials: the SPA sends every request with
		// credentials included (session cookie), and browsers reject
		// credentialed responses that lack this header — even when auth is
		// off. Safe because origins are an explicit allowlist, never "*".
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		// Always available so the frontend can discover the active features.
		r.Get("/config", configHandler(feats))

		r.Route("/auth", func(r chi.Router) {
			r.Get("/notion/login", a.BeginLogin)
			r.Get("/notion/callback", a.HandleCallback)
			r.Post("/login", a.PasswordLogin)
			r.Get("/me", meHandler(a))
			r.Post("/logout", a.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(a.Middleware)
			mountResources(r, s, q, feats, a, y)
		})
	})

	return r
}

func mountResources(r chi.Router, s *store.Store, q *quotes.Service, feats config.Features, a *auth.Service, y *notion.Syncer) {
	if feats.Enabled(config.Income) {
		crud[domain.Income]{
			list: s.ListIncomes, create: s.CreateIncome,
			update: s.UpdateIncome, delete: s.DeleteIncome,
		}.mount(r, "/incomes")
	}

	if feats.Enabled(config.Expenses) {
		crud[domain.Expense]{
			list: s.ListExpenses, create: s.CreateExpense,
			update: s.UpdateExpense, delete: s.DeleteExpense,
		}.mount(r, "/expenses")
	}

	if feats.Enabled(config.Investments) {
		crud[domain.Investment]{
			list: s.ListInvestments, create: s.CreateInvestment,
			update: s.UpdateInvestment, delete: s.DeleteInvestment,
		}.mount(r, "/investments")

		r.Route("/quotes", func(r chi.Router) {
			r.Post("/refresh", refreshPricesHandler(q))
			r.Get("/search/{kind}", searchHandler(q))
		})
	}

	if feats.Enabled(config.Categories) {
		crud[domain.Category]{
			list: s.ListCategories, create: s.CreateCategory,
			update: s.UpdateCategory, delete: s.DeleteCategory,
		}.mount(r, "/categories")
	}

	if feats.Enabled(config.Backup) {
		r.Get("/backup/export", exportHandler(s))
		r.Post("/backup/import", importHandler(s))
	}

	r.Route("/notion", func(r chi.Router) {
		r.Get("/status", notionStatusHandler(a, y))
		r.Post("/sync", notionSyncHandler(a, y))
		r.Post("/pull", notionPullHandler(a, y))
	})
}

func configHandler(feats config.Features) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"app":      store.AppName,
			"features": feats.List(),
		})
	}
}

func refreshPricesHandler(q *quotes.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		writeJSON(w, http.StatusOK, q.RefreshAll(ctx))
	}
}

func searchHandler(q *quotes.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind := chi.URLParam(r, "kind")
		query := r.URL.Query().Get("q")
		if len(query) < 2 {
			writeJSON(w, http.StatusOK, []quotes.SearchHit{})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		hits, err := q.Search(ctx, kind, query)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, hits)
	}
}

func exportHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		snap := s.Export()
		filename := fmt.Sprintf("moneytracker-backup-%s.json", time.Now().Format("2006-01-02"))
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		writeJSON(w, http.StatusOK, snap)
	}
}

func importHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var snap store.Snapshot
		if err := decodeJSON(r, &snap); err != nil {
			writeError(w, http.StatusBadRequest, "invalid snapshot JSON: "+err.Error())
			return
		}
		if snap.App != store.AppName {
			writeError(w, http.StatusUnprocessableEntity, "not a money-tracker snapshot")
			return
		}
		if snap.SchemaVersion > store.SchemaVersion {
			writeError(w, http.StatusUnprocessableEntity, "snapshot is from a newer app version")
			return
		}
		if err := s.Import(snap); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "imported"})
	}
}

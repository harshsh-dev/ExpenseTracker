package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"moneytracker/backend/internal/domain"
	"moneytracker/backend/internal/quotes"
	"moneytracker/backend/internal/store"
)

// NewRouter builds the HTTP handler for the API.
func NewRouter(s *store.Store, q *quotes.Service, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		crud[domain.Income]{
			list: s.ListIncomes, create: s.CreateIncome,
			update: s.UpdateIncome, delete: s.DeleteIncome,
		}.mount(r, "/incomes")

		crud[domain.Expense]{
			list: s.ListExpenses, create: s.CreateExpense,
			update: s.UpdateExpense, delete: s.DeleteExpense,
		}.mount(r, "/expenses")

		crud[domain.Investment]{
			list: s.ListInvestments, create: s.CreateInvestment,
			update: s.UpdateInvestment, delete: s.DeleteInvestment,
		}.mount(r, "/investments")

		crud[domain.Category]{
			list: s.ListCategories, create: s.CreateCategory,
			update: s.UpdateCategory, delete: s.DeleteCategory,
		}.mount(r, "/categories")

		r.Get("/backup/export", exportHandler(s))
		r.Post("/backup/import", importHandler(s))

		r.Route("/quotes", func(r chi.Router) {
			r.Post("/refresh", refreshPricesHandler(q))
			r.Get("/search/{kind}", searchHandler(q))
		})
	})

	return r
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

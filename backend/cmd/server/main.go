package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"moneytracker/backend/internal/api"
	"moneytracker/backend/internal/auth"
	"moneytracker/backend/internal/config"
	"moneytracker/backend/internal/notion"
	"moneytracker/backend/internal/quotes"
	"moneytracker/backend/internal/storage"
	"moneytracker/backend/internal/store"
)

func main() {
	port := env("PORT", "8080")
	origins := strings.Split(env("ALLOWED_ORIGINS", "http://localhost:5173"), ",")
	feats := config.Parse(env("FEATURES", "all"))

	snapBlob, authBlob := storageBlobs()

	s, err := store.New(snapBlob)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	log.Printf("features enabled: %s", feats)

	// Notion login ("Sign in with Notion"). Optional: without client
	// credentials the app runs open, exactly as before.
	authCfg := auth.Config{
		ClientID:     env("NOTION_CLIENT_ID", ""),
		ClientSecret: env("NOTION_CLIENT_SECRET", ""),
		// Default assumes the Vite dev proxy so the session cookie lands
		// on the frontend origin; set explicitly in production.
		RedirectURI:   env("NOTION_REDIRECT_URI", "http://localhost:5173/api/auth/notion/callback"),
		FrontendURL:   env("FRONTEND_URL", "/"),
		SessionSecret: env("SESSION_SECRET", ""),
		AllowedEmails: strings.Split(env("ALLOWED_NOTION_EMAILS", ""), ","),
		CrossSite:     env("CROSS_SITE_COOKIES", "off") == "on",
	}
	a, err := auth.New(authCfg, authBlob)
	if err != nil {
		log.Fatalf("init auth: %v", err)
	}
	if a.Enabled() {
		log.Println("notion login enabled")
	} else {
		log.Println("notion login not configured (NOTION_CLIENT_ID/SECRET unset) — running open")
	}
	y := notion.NewSyncer(s, a.Accounts())

	q := quotes.New(s)
	if feats.Enabled(config.Investments) {
		startPriceScheduler(q)
	}

	srv := &http.Server{
		Addr:         announceAddr(port),
		Handler:      api.NewRouter(s, q, feats, origins, a, y),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// startPriceScheduler refreshes investment prices shortly after boot and then
// on an interval (default 12h). Cadence is low on purpose: NAVs are daily and
// free sources (esp. NSE) must not be hammered. Set QUOTES_REFRESH=off to skip.
func startPriceScheduler(q *quotes.Service) {
	if env("QUOTES_REFRESH", "on") == "off" {
		return
	}
	interval, err := time.ParseDuration(env("QUOTES_REFRESH_INTERVAL", "12h"))
	if err != nil {
		interval = 12 * time.Hour
	}
	go func() {
		time.Sleep(10 * time.Second)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			res := q.RefreshAll(ctx)
			cancel()
			log.Printf("price refresh: %d investments processed", len(res.Results))
			time.Sleep(interval)
		}
	}()
}

// storageBlobs picks the persistence backend: local files by default, or
// Firestore (STORAGE_BACKEND=firestore) for hosts without a persistent disk.
func storageBlobs() (snapshot, accounts storage.Blob) {
	switch env("STORAGE_BACKEND", "file") {
	case "firestore":
		fs, err := storage.NewFirestore(
			context.Background(),
			env("FIRESTORE_PROJECT_ID", ""),
			[]byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")),
		)
		if err != nil {
			log.Fatalf("init firestore: %v", err)
		}
		log.Println("storage: firestore")
		return fs.Blob("snapshot.json"), fs.Blob("auth.json")
	default:
		dataPath := env("DATA_PATH", "data/snapshot.json")
		authPath := env("AUTH_PATH", filepath.Join(filepath.Dir(dataPath), "auth.json"))
		log.Printf("storage: file (snapshot: %s)", dataPath)
		return storage.NewFileBlob(dataPath), storage.NewFileBlob(authPath)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func announceAddr(port string) string {
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

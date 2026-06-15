package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"moneytracker/backend/internal/api"
	"moneytracker/backend/internal/quotes"
	"moneytracker/backend/internal/store"
)

func main() {
	port := env("PORT", "8080")
	dataPath := env("DATA_PATH", "data/snapshot.json")
	origins := strings.Split(env("ALLOWED_ORIGINS", "http://localhost:5173"), ",")

	s, err := store.New(dataPath)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	log.Printf("store ready (snapshot: %s)", dataPath)

	q := quotes.New(s)
	startPriceScheduler(q)

	srv := &http.Server{
		Addr:         announceAddr(port),
		Handler:      api.NewRouter(s, q, origins),
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

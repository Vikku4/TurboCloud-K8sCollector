package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/turbocloud-io/k8s-collector/internal/config"
	"github.com/turbocloud-io/k8s-collector/internal/core"
	"github.com/turbocloud-io/k8s-collector/internal/httpserver"
	tclog "github.com/turbocloud-io/k8s-collector/internal/log"
	"github.com/turbocloud-io/k8s-collector/internal/store"
)

func main() {
	// 1. Load .env file for local dev (if present)
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, falling back to OS environment")
	}

	// 2. Load config from env
	cfg := config.Load()
	logger := tclog.New(cfg.LogLevel)

	ctx := context.Background()

	// 3. Open Postgres connection
	db, err := store.OpenPostgres(ctx, cfg.DBDSN)
	if err != nil {
		logger.Errorf("db connection error: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// 4. Wire store -> service -> http server
	st := store.NewPostgresStore(db)
	svc := core.NewService(st)
	srvHTTP := httpserver.NewServer(svc, cfg.RequestTimeout, logger)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srvHTTP.Router(),
	}

	// 5. Start HTTP server
	go func() {
		logger.Infof("k8s-collector listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("http server error: %v", err)
			os.Exit(1)
		}
	}()

	// 6. Graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Infof("shutting down...")
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Errorf("server shutdown error: %v", err)
	}
}

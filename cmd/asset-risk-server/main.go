package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"asset-risk-system/internal/httpapi"
	"asset-risk-system/internal/store"
)

func main() {
	var (
		addr     = flag.String("addr", envOr("ASSET_RISK_ADDR", ":8080"), "HTTP listen address")
		dataPath = flag.String("data", envOr("ASSET_RISK_DATA", "data/assets.json"), "JSON data file")
		webDir   = flag.String("web", envOr("ASSET_RISK_WEB", ""), "directory containing built frontend assets")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	repository, err := store.New(*dataPath)
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           httpapi.NewWithStatic(repository, logger, *webDir),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("asset risk server listening", "addr", *addr, "data", *dataPath, "web", *webDir)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

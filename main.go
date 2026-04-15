package main

import (
	"embed"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"parse-api-messages/pkg/auth"
	"parse-api-messages/pkg/waviot"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: resolveLogLevel()}))
	slog.SetDefault(logger)
	slog.Info("app starting")

	jwt, err := auth.NewFileProvider()
	if err != nil {
		slog.Error("auth provider init failed", "err", err)
		os.Exit(1)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	wvc := waviot.NewClient(httpClient, jwt)

	app := NewApp(jwt, wvc)

	if err := wails.Run(&options.App{
		Title:  "Parse API Messages",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		slog.Error("wails run failed", "err", err)
	}
}

// resolveLogLevel читает LOG_LEVEL из env. По умолчанию — debug в dev.
func resolveLogLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

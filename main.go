package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/disgoorg/disgo/handler"

	"github.com/speedtest002/azunyan-2-go/azunyan2go"
	"github.com/speedtest002/azunyan-2-go/commands"
	"github.com/speedtest002/azunyan-2-go/db"
	"github.com/speedtest002/azunyan-2-go/handlers"
	"github.com/speedtest002/azunyan-2-go/middleware"
	"github.com/speedtest002/azunyan-2-go/service"
)

var (
	Version = "dev"
	Commit  = "xxx"
)

func main() {
	shouldSyncCommands := flag.Bool("sync-commands", false, "Whether to sync commands to discord")
	path := flag.String("config", "config.toml", "path to config")
	flag.Parse()

	cfg, err := azunyan2go.LoadConfig(*path)
	if err != nil {
		slog.Error("Failed to read config", slog.Any("err", err))
		os.Exit(-1)
	}

	setupLogger(cfg.Log)
	slog.Info("Starting azunyan-2-go...", slog.String("version", Version), slog.String("commit", Commit))
	slog.Info("Syncing commands", slog.Bool("sync", *shouldSyncCommands))

	database, err := db.Connect(cfg.Database.User, cfg.Database.Password, cfg.Database.Dsn)
	if err != nil {
		slog.Error("DB connect failed", "err", err)
		os.Exit(-1)
	}
	slog.Info("Connected to DB")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := database.Migrate(ctx); err != nil {
		slog.Error("DB migration failed", "err", err)
		os.Exit(-1)
	}

	database.StartKeepAlive(ctx)

	b := azunyan2go.New(*cfg, Version, Commit)
	b.DB = database

	vietQR, err := service.NewVietQRService()
	if err != nil {
		slog.Error("Failed to fetch banks from VietQR", "err", err)
	}
	b.VietQR = vietQR

	h := handler.New()
	h.Use(middleware.Recover, middleware.LoggingMiddleware)

	if err = b.SetupBot(h); err != nil {
		slog.Error("Failed to setup bot", slog.Any("err", err))
		os.Exit(-1)
	}

	b.Scheduler = service.NewScheduler(b.DB, b.Client, 30*time.Second)
	b.Scheduler.Start()

	commands.Register(h, b)
	handlers.Register(b)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		b.Client.Close(shutdownCtx)
		b.Scheduler.Shutdown()
		b.DB.Close()
	}()

	if *shouldSyncCommands {
		slog.Info("Syncing commands", slog.Any("guild_ids", cfg.Bot.DevGuilds))
		if err = handler.SyncCommands(&b.Client, commands.Commands, cfg.Bot.DevGuilds); err != nil {
			slog.Error("Failed to sync commands", slog.Any("err", err))
		}
	}

	gatewayCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = b.Client.OpenGateway(gatewayCtx); err != nil {
		slog.Error("Failed to open gateway", slog.Any("err", err))
		os.Exit(-1)
	}

	slog.Info("Bot is running. Press CTRL-C to exit.")
	<-ctx.Done()
	slog.Info("Shutting down bot...")
}

func setupLogger(cfg azunyan2go.LogConfig) {
	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     cfg.Level,
	}

	var sHandler slog.Handler
	switch cfg.Format {
	case "json":
		sHandler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		sHandler = slog.NewTextHandler(os.Stdout, opts)
	default:
		slog.Error("Unknown log format", slog.String("format", cfg.Format))
		os.Exit(-1)
	}
	slog.SetDefault(slog.New(sHandler))
}

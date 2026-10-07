package main

import (
	"LazyCatBot/internal/config"
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"LazyCatBot/internal/storage/migrations"
	"LazyCatBot/internal/worker"
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	slogadapter "github.com/axiomhq/axiom-go/adapters/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	//env, Axiom logs
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	file, err := os.OpenFile("bot.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Log read file error: %v", err)
	}
	defer file.Close()

	axiomHandler, err := slogadapter.New()
	if err != nil {
		log.Fatalf("Failed to create Axiom handler: %v", err)
	}
	defer axiomHandler.Close()

	appCfg := config.LoadConfig()

	fileHandler := slog.NewJSONHandler(file, nil)
	handlers := []slog.Handler{fileHandler, axiomHandler}

	if appCfg.IsDebug {
		consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		handlers = append(handlers, consoleHandler)
		log.Println("=== RUNNING IN DEBUG MODE ===")
	}

	logger := slog.New(NewMultiHandler(handlers...))
	slog.SetDefault(logger)

	//Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Discord token and session
	if appCfg.DiscordToken == "" {
		slog.Error("DISCORD_TOKEN doesn`t set")
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + appCfg.DiscordToken)
	if err != nil {
		slog.Error("DiscordGo session create error", "error", err)
		os.Exit(1)
	}

	//DB
	db, err := sql.Open("sqlite", "bot.db")
	if err != nil {
		slog.Error("Failed to init db", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA busy_timeout=10000;")
	db.SetMaxOpenConns(1)
	goose.SetBaseFS(migrations.EmbedFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		slog.Error("Failed to set goose dialect", "error", err)
		os.Exit(1)
	}
	slog.Info("Running database migrations...")
	if err := goose.Up(db, "."); err != nil {
		slog.Error("Database migrations failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database migrations applied successfully!")

	//Storage
	lbStore := storage.NewLeaderboardStorage(db, logger)
	subStore := storage.NewSubscribeStorage(db)
	gmStore := storage.NewGuildMembersStorage(db)
	pSubStore := storage.NewPlayerSubscribeStorage(db)
	ccStore := storage.NewCharacterCacheStorage(db)
	arStore := storage.NewActualRaidsStorage(db)

	sirusClient := sirus.NewClient(logger, appCfg.GetSirusURLs())
	discordReporter := discord.NewDiscordReporter(dg)

	killWorker := worker.NewWorker(
		sirusClient,
		lbStore, subStore,
		gmStore,
		pSubStore,
		ccStore,
		arStore,
		discordReporter,
		logger,
		appCfg,
	)

	killWorker.StartCronScheduler()
	go killWorker.StartProcessor(ctx)
	go killWorker.GuildKillMonitor(ctx)
	go killWorker.PlayerKillMonitor(ctx)
	go killWorker.MythicRunsMonitor(ctx)
	go killWorker.GuildMembersUpdater(ctx)
	go killWorker.CharacterCacheUpdater(ctx)
	go killWorker.CleanupRoutine(ctx)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	h := discord.NewHandler(sirusClient, lbStore, subStore, pSubStore, gmStore, arStore, logger)

	dg.AddHandler(h.InteractionCreate)
	if !appCfg.IsDebug {
		dg.AddHandler(h.GuildCreate)
	} else {
		slog.Info("[DEBUG MODE] Skip GuildCreate handler registration")
	}

	err = dg.Open()
	if err != nil {
		slog.Error("Discord connection error", "error", err)
		os.Exit(1)
	}
	defer dg.Close()

	manageChannelsPerm := int64(discordgo.PermissionManageChannels)

	commands := []*discordgo.ApplicationCommand{
		{
			Name:                     "menu",
			Description:              "Открыть меню настройки трекинга и отправки автоматических отчетов",
			DefaultMemberPermissions: &manageChannelsPerm,
		},
		{
			Name:        "topm",
			Description: "Открыть меню отправки рейтингов по босам среди игроков гильдии",
		},
		{
			Name:        "help",
			Description: "Показать справку по боту",
		},
		{
			Name:        "list",
			Description: "Показать список отслеживаемых гильдий в текущем канале",
		},
		{
			Name:        "listcats",
			Description: "Список отслеживаемых игроков в данном канале",
		},
	}

	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)
	if err != nil {
		slog.Error("Error registering commands", "error", err)
		os.Exit(1)
	}

	fmt.Println("Bot is running. Slash Commands registered. Ctrl+C exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	slog.Info("Shutting down gracefully...")
	cancel()

	time.Sleep(5 * time.Second)
	slog.Info("Bot has been stopped")
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &MultiHandler{handlers: handlers}
}
func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}
func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		_ = h.Handle(ctx, r)
	}
	return nil
}
func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}
func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}

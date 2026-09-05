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

	fileHandler := slog.NewJSONHandler(file, nil)
	logger := slog.New(NewMultiHandler(fileHandler, axiomHandler))
	slog.SetDefault(logger)

	//Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Discord token and session
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		slog.Error("DISCORD_TOKEN doesn`t set")
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + token)
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
	db.Exec("PRAGMA busy_timeout=5000;")
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
	arStore := storage.NewActualRaidsStorage(db)

	appCfg, err := config.LoadConfig("config.json")
	if err != nil {
		slog.Warn("Failed to load config.json, using defaults", "error", err)
	}

	sirusClient := sirus.NewClient(logger, appCfg.GetSirusURLs())
	discordReporter := discord.NewDiscordReporter(dg)

	killWorker := worker.NewWorker(sirusClient, lbStore, subStore, gmStore, pSubStore, arStore, discordReporter, logger)

	killWorker.StartCronScheduler()
	go killWorker.StartProcessor(ctx)
	go killWorker.GuildKillMonitor(ctx)
	go killWorker.PlayerKillMonitor(ctx)
	go killWorker.GuildMembersUpdater(ctx)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	h := discord.NewHandler(sirusClient, lbStore, subStore, pSubStore, gmStore, arStore, logger)

	dg.AddHandler(h.InteractionCreate)
	dg.AddHandler(h.GuildCreate)

	err = dg.Open()
	if err != nil {
		slog.Error("Discord connection error", "error", err)
		os.Exit(1)
	}
	defer dg.Close()

	manageChannelsPerm := int64(discordgo.PermissionManageChannels)

	realmChoices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Nevermine x3 (По умолчанию)", Value: "x3"},
		{Name: "Soulseeker x1", Value: "x1"},
		{Name: "Scourge x2", Value: "x2"},
		{Name: "Sirus x5", Value: "x5"},
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:                     "set",
			Description:              "Подписаться на отчеты гильдии в этом канале",
			DefaultMemberPermissions: &manageChannelsPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "ID гильдии на Сирусе",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "realm",
					Description: "Игровой сервер (по умолчанию х3)",
					Required:    false,
					Choices:     realmChoices,
				},
			},
		},
		{
			Name:                     "unset",
			Description:              "Отписаться от отчетов гильдии в этом канале",
			DefaultMemberPermissions: &manageChannelsPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "ID гильдии на Сирусе",
					Required:    true,
				},
			},
		},
		{
			Name:                     "setcat",
			Description:              "Добавить трекинг игрока в этом канале",
			DefaultMemberPermissions: &manageChannelsPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Имя игрока",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "realm",
					Description: "Игровой сервер (по умолчанию х3)",
					Required:    false,
					Choices:     realmChoices,
				},
			},
		},
		{
			Name:                     "unsetcat",
			Description:              "Отписаться от отчетов игроков в этом канале",
			DefaultMemberPermissions: &manageChannelsPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "ID игрока на Сирусе",
					Required:    true,
				},
			},
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
		{
			Name:                     "menu",
			Description:              "Открыть меню настройки трекинга и отправки автоматических отчетов",
			DefaultMemberPermissions: &manageChannelsPerm,
		},
		{
			Name:        "topm",
			Description: "Открыть меню отправки рейтингов по босам среди игроков гильдии",
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

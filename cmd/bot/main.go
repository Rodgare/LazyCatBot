package main

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"LazyCatBot/internal/worker"
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	slogadapter "github.com/axiomhq/axiom-go/adapters/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
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

	db, err := sql.Open("sqlite", "bot.db")
	if err != nil {
		slog.Error("Failed to init db", "error", err)
		os.Exit(1)
	}
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA busy_timeout=5000;")
	defer db.Close()

	lbStore := storage.NewLeaderboardStorage(db, logger)
	subStore := storage.NewSubscribeStorage(db)
	gmStore := storage.NewGuildMembersStorage(db)
	pSubStore := storage.NewPlayerSubscribeStorage(db)
	lbStore.InitDB()
	subStore.InitDB()
	gmStore.InitDB()
	pSubStore.InitDB()

	sirusClient := sirus.NewClient(logger)

	killWorker := worker.NewWorker(sirusClient, lbStore, subStore, gmStore, pSubStore, dg, logger)
	killWorker.StartCronScheduler()
	go killWorker.StartProcessor()
	go killWorker.GuildKillMonitor()
	go killWorker.PlayerKillMonitor()

	go killWorker.GuildMembersUpdater()

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	h := discord.NewHandler(sirusClient, lbStore, subStore, pSubStore, gmStore, logger)

	dg.AddHandler(h.InteractionCreate)
	dg.AddHandler(h.GuildCreate)

	err = dg.Open()
	if err != nil {
		slog.Error("Discord connection error", "error", err)
		os.Exit(1)
	}
	defer dg.Close()

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "set",
			Description: "Подписаться на отчеты гильдии в этом канале",
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
			Name:        "unset",
			Description: "Отписаться от отчетов гильдии в этом канале",
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
			Name:        "setcat",
			Description: "Добавить трекинг игрока в этом канале",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Имя игрока на Сервере х3",
					Required:    true,
				},
			},
		},
		{
			Name:        "unsetcat",
			Description: "Отписаться от отчетов игроков в этом канале",
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
			Name:        "menu",
			Description: "Открыть меню настройки трекинга и отправки автоматических отчетов",
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

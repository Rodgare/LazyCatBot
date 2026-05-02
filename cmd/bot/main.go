package main

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/storage"
	"LazyCatBot/internal/worker"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	file, err := os.OpenFile("bot.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Log read file error: %v", err)
	}
	defer file.Close()
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime)

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error load .env file")
	}
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN doesn`t set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("DiscordGo session create error: %v", err)
		return
	}

	db, err := sql.Open("sqlite", "bot.db")
	if err != nil {
		log.Fatal(err)
	}
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA busy_timeout=5000;")
	defer db.Close()

	lbStore := storage.NewLeaderboardStorage(db)
	subStore := storage.NewSubscribeStorage(db)
	gmStore := storage.NewGuildMembersStorage(db)
	pSubStore := storage.NewPlayerSubscribeStorage(db)
	lbStore.InitDB()
	subStore.InitDB()
	gmStore.InitDB()
	pSubStore.InitDB()

	// go worker.StartLeaderboardSync(lbStore)
	// go worker.StartMetasirusLbSync(lbStore)
	worker.StartCronScheduler(lbStore)

	killWorker := worker.NewWorker(lbStore, subStore, pSubStore, dg)
	go killWorker.StartProcessor()
	go killWorker.GuildKillMonitor()
	go killWorker.PlayerKillMonitor()
	// go worker.KillMonitor(lbStore, subStore, playerSubStore, dg)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	h := &discord.BotHandler{
		LbStore:        lbStore,
		SubStore:       subStore,
		PlayerSubStore: pSubStore,
	}
	dg.AddHandler(h.InteractionCreate)
	dg.AddHandler(h.GuildCreate)

	err = dg.Open()
	if err != nil {
		log.Fatalf("Discord connection error: %v", err)
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
	}

	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)
	if err != nil {
		log.Fatalf("Error registering commands: %v", err)
	}

	fmt.Println("Bot is running. Slash Commands registered. Ctrl+C exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

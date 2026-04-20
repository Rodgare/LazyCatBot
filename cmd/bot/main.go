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
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error load .env file")
	}
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN doesn`t set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("DiscordGo session create error:", err)
		return
	}

	db, err := sql.Open("sqlite", "bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	lbStore := storage.NewLeaderboardStorage(db)
	subStore := storage.NewSubscribeStorage(db)
	lbStore.InitDB()
	subStore.InitDB()

	go worker.StartLeaderboardSync(lbStore)
	go worker.KillMonitor(lbStore, subStore, dg)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	h := &discord.BotHandler{
		LbStore:  lbStore,
		SubStore: subStore,
	}
	dg.AddHandler(h.InteractionCreate)
	dg.AddHandler(h.GuildCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Connection error:", err)
		return
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
			Name:        "help",
			Description: "Показать справку по боту",
		},
	}

	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)
	if err != nil {
		log.Printf("Error registering commands: %v", err)
	}

	fmt.Println("Bot is running. Slash Commands registered. Ctrl+C exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

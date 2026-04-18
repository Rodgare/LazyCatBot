package main

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/storage"
	"LazyCatBot/internal/worker"
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

	leaderboardStore := storage.NewLeaderboardStorage()
	bossKillsStore := storage.NewBossKillsStorage()

	go worker.StartLeaderboardSync(leaderboardStore)
	go worker.KillMonitor(leaderboardStore, bossKillsStore, dg)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	h := &discord.BotHandler{
		Store: leaderboardStore,
	}
		Store: leaderboardStore,
	}
	dg.AddHandler(h.MessageCreate)
	err = dg.Open()

	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}

	defer dg.Close()

	fmt.Println("Bot is running. Ctrl+C exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

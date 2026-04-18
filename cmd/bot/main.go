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
		log.Fatal("Ошибка загрузки .env файла")
	}
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN не установлен")
	}
		log.Fatal("DISCORD_TOKEN не установлен")
	}

	leaderboardStore := storage.NewLeaderboardStorage()
	bossKillsStore := storage.NewBossKillsStorage()

	go worker.StartLeaderboardSync(leaderboardStore)
	go worker.KillMonitor(bossKillsStore)

	dg, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Println("Ошибка создания сессии:", err)
		return
	}
		fmt.Println("Ошибка создания сессии:", err)
		return
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	h := &discord.BotHandler{
		Store: leaderboardStore,
	}
		Store: leaderboardStore,
	}
	dg.AddHandler(h.MessageCreate)
	err = dg.Open()

	if err != nil {
		fmt.Println("Ошибка открытия соединения:", err)
		return
	}
	if err != nil {
		fmt.Println("Ошибка открытия соединения:", err)
		return
	}

	defer dg.Close()

	fmt.Println("Бот запущен. Ctrl+C для выхода.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

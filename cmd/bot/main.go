package main

import (
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

var leaderboardStore *storage.LeaderboardStorage

func main() {
	leaderboardStore = storage.NewLeaderboardStorage()

	go worker.StartLeaderboardSync(leaderboardStore)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}
	token := os.Getenv("DISCORD_TOKEN")

	dg, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Println("Ошибка создания сессии:", err)
		return
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// dg.AddHandler(messageCreate)

	err = dg.Open()

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

// func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
//     if m.Author.ID == s.State.User.ID {
//         return
//     }

//     if strings.ToLower(m.Content) == "/топ" {
//         table := discord.FormatTotalTopGuild("Coda", globalStore.ActiveRaids)

//         _, err := s.ChannelMessageSend(m.ChannelID, table)
//         if err != nil {
//             fmt.Println("Ошибка отправки сообщения:", err)
//         }
//     }
// }

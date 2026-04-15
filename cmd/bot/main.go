package main

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var globalStore storage.RaidStorage

func main() {
	raids, err := sirus.GetActualRaids()
	if err != nil {
		log.Printf("Ошибка получения рейдов: %v", err)
	}

	globalStore = storage.RaidStorage{
		ActiveRaids: raids,
	}

	store := storage.RaidStorage{
		ActiveRaids: raids,
	}
	fmt.Printf("Содержимое store.ActiveRaids: %#v\n", store.ActiveRaids)
	fmt.Printf("Загружено актуальных рейдов: %d\n", len(store.ActiveRaids))

	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))

	if err != nil {
		fmt.Println("Ошибка создания сессии:", err)
		return
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	dg.AddHandler(messageCreate)

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

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.ToLower(m.Content) == "/топ" {
		table := discord.FormatTotalTopGuild("Coda", globalStore.ActiveRaids)

		_, err := s.ChannelMessageSend(m.ChannelID, table)
		if err != nil {
			fmt.Println("Ошибка отправки сообщения:", err)
		}
	}
}

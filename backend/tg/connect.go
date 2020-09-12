// Package tg handle work with tg
package tg

import (
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Connect - connection to a tg
func Connect(telegramToken, telegramChatID string) (bot *tgbotapi.BotAPI, chatID int64) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatalf("[ERROR] telegram api, %v", err)
	}
	chatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		log.Fatalf("[ERROR] chat id, %v", err)
	}
	return
}

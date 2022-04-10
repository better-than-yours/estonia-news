package telegram

import (
	"strconv"

	"estonia-news/misc"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Connect do connection to telegram
func Connect(telegramToken, telegramChatID string) (bot *tgbotapi.BotAPI, chatID int64) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		misc.Fatal("tg_api", "telegram api", err)
	}
	chatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		misc.Fatal("tg_chat_id", "chat id", err)
	}
	return
}

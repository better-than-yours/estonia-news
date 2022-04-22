package telegram

import (
	"estonia-news/misc"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Connect do connection to telegram
func Connect(telegramToken string) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		misc.Fatal("tg_api", "telegram api", err)
	}
	return bot
}

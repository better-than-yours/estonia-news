package telegram

import (
	"strconv"

	"estonia-news/misc"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/prometheus/client_golang/prometheus"
)

// Connect do connection to telegram
func Connect(telegramToken, telegramChatID string) (bot *tgbotapi.BotAPI, chatID int64) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		misc.TaskErrors.With(prometheus.Labels{"error": "tg_api"}).Inc()
		misc.PushMetrics()
		misc.L.Logf("FATAL telegram api, %v", err)
	}
	chatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		misc.TaskErrors.With(prometheus.Labels{"error": "tg_chat_id"}).Inc()
		misc.PushMetrics()
		misc.L.Logf("FATAL chat id, %v", err)
	}
	return
}

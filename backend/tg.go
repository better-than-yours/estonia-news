package main

import (
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/prometheus/client_golang/prometheus"
)

func connectTg(telegramToken, telegramChatID string) (bot *tgbotapi.BotAPI, chatID int64) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		taskErrors.With(prometheus.Labels{"error": "tg_api"}).Inc()
		pushMetrics()
		l.Logf("FATAL telegram api, %v", err)
	}
	chatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		taskErrors.With(prometheus.Labels{"error": "tg_chat_id"}).Inc()
		pushMetrics()
		l.Logf("FATAL chat id, %v", err)
	}
	return
}

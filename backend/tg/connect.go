// Package tg handle work with tg
package tg

import (
	"strconv"

	"github.com/go-pkgz/lgr"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var l = lgr.New(lgr.Msec, lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

// Connect - connection to a tg
func Connect(telegramToken, telegramChatID string) (bot *tgbotapi.BotAPI, chatID int64) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		l.Logf("FATAL telegram api, %v", err)
	}
	chatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		l.Logf("FATAL chat id, %v", err)
	}
	return
}

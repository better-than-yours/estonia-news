package config

import (
	"time"

	"estonia-news/entity"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

// TimeoutBetweenLoops is main loop timeout
var TimeoutBetweenLoops = 5 * time.Minute

// TimeoutBetweenMessages is timeout between attempts to send a message
var TimeoutBetweenMessages = 1 * time.Second

// TimeShift is time shift
var TimeShift = 1

// Params is params struct
type Params struct {
	Bot               *tgbotapi.BotAPI
	DB                *gorm.DB
	Feed              *gofeed.Feed
	Item              *gofeed.Item
	Provider          entity.Provider
	ChatID            int64
	Lang              string
	BlockedCategories []string
	BlockedWords      []string
}

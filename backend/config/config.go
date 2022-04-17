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
var TimeoutBetweenMessages = time.Second

// TimeShift get messages from the last hours
var TimeShift = time.Hour

// PurgeOldEntriesEvery is time for purge old entries
var PurgeOldEntriesEvery = time.Hour

// Params is params struct
type Params struct {
	Bot               *tgbotapi.BotAPI
	DB                *gorm.DB
	Feed              *gofeed.Feed
	Provider          entity.Provider
	ChatID            int64
	Lang              string
	BlockedCategories []string
	BlockedWords      []string
}

// FeedItem is feed item struct
type FeedItem struct {
	GUID          string
	Link          string
	Title         string
	Description   string
	Published     string
	Categories    []string
	CategoriesIds []int
}

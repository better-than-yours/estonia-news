package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"

	"github.com/better-than-yours/estonia-news/db"
	"github.com/better-than-yours/estonia-news/http"
	"github.com/better-than-yours/estonia-news/tg"
)

// TimeoutBetweenLoops - Main loop timeout
var TimeoutBetweenLoops = 5 * time.Minute

// TimeoutBetweenMessages - timeout between attempts to send a message
var TimeoutBetweenMessages = 5 * time.Second

// CheckFromTime - time ago from which should check messages
var CheckFromTime = 1 * time.Hour

// Message - config
type Message struct {
	FeedTitle   string
	Title       string
	Description string
	Categories  []string
	Link        string
	ImageURL    string
}

// Params - params
type Params struct {
	Bot               *tgbotapi.BotAPI
	DB                *gorm.DB
	Feed              *gofeed.Feed
	Item              *gofeed.Item
	Provider          db.Provider
	ChatID            int64
	Lang              string
	BlockedCategories []string
}

func getImageURL(item *gofeed.Item) string {
	var url string = ""
	if len(item.Enclosures) > 0 && item.Enclosures[0].URL != "" {
		url = item.Enclosures[0].URL
	} else if len(item.Extensions["media"]["thumbnail"]) > 0 && item.Extensions["media"]["thumbnail"][0].Attrs["url"] != "" {
		url = item.Extensions["media"]["thumbnail"][0].Attrs["url"]
	}
	return url
}

func formatText(text string) string {
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`([\x{0020}\x{00a0}\x{1680}\x{180e}\x{2000}-\x{200b}\x{202f}\x{205f}\x{3000}\x{feff}])`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`(\s+)`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`(\n)\n+`).ReplaceAllString(text, "$1")
	return text
}

func getText(params *Params, msg *Message) (title, description string) {
	title = formatText(msg.Title)
	description = formatText(msg.Description)
	if params.Provider.Lang == "EST" {
		if title != "" {
			text, err := http.Translate(title, "et", "en")
			if err != nil {
				log.Fatalf("[ERROR] get translate, %v (%s)", err, title)
			}
			title = text
		}
		if description != "" {
			text, err := http.Translate(description, "et", "en")
			if err != nil {
				log.Fatalf("[ERROR] get translate, %v (%s)", err, description)
			}
			description = text
		}
	}
	return
}

func renderMessageBlock(msg *Message, title, description string) string {
	return fmt.Sprintf("<b>%s</b>\n\n%s\n\n<a href=\"%s\">%s</a>", title, description, msg.Link, strings.TrimSpace(msg.FeedTitle))
}

func createNewMessageObject(params *Params, msg *Message) (*tgbotapi.PhotoConfig, error) {
	content, err := http.GetImage(msg.ImageURL)
	if err != nil {
		return nil, err
	}
	file := tgbotapi.FileBytes{Name: msg.ImageURL, Bytes: content}
	title, description := getText(params, msg)
	text := renderMessageBlock(msg, title, description)
	return &tgbotapi.PhotoConfig{
		BaseFile: tgbotapi.BaseFile{
			BaseChat:    tgbotapi.BaseChat{ChatID: params.ChatID},
			File:        file,
			UseExisting: false,
		},
		Caption:   text,
		ParseMode: tgbotapi.ModeHTML,
	}, nil
}

func createEditMessageObject(params *Params, messageID int, msg *Message) *tgbotapi.EditMessageCaptionConfig {
	title, description := getText(params, msg)
	text := renderMessageBlock(msg, title, description)
	return &tgbotapi.EditMessageCaptionConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:    params.ChatID,
			MessageID: messageID,
		},
		Caption:   text,
		ParseMode: tgbotapi.ModeHTML,
	}
}

func createDeleteMessageObject(params *Params, messageID int) *tgbotapi.DeleteMessageConfig {
	return &tgbotapi.DeleteMessageConfig{
		ChatID:    params.ChatID,
		MessageID: messageID,
	}
}

func hasChanges(item *gofeed.Item, entry db.Entry) bool {
	if entry.Title != item.Title || entry.Description != item.Description || entry.Link != item.Link {
		return true
	}
	return false
}

func addRecord(params *Params, entries *[]db.Entry) error {
	var Item = params.Item
	for _, entry := range *entries {
		if entry.GUID == Item.GUID {
			if hasChanges(Item, entry) {
				log.Printf("[INFO] send edit item with guid: %v", Item.GUID)
				if editMessageErr := editMessage(params, entry); editMessageErr != nil {
					return editMessageErr
				}
			}
			return nil
		}
	}
	log.Printf("[INFO] send new item with guid: %v", Item.GUID)
	err := addMessage(params)
	if err != nil {
		return err
	}
	time.Sleep(TimeoutBetweenMessages)
	return nil
}

func deleteRecord(params *Params, entry db.Entry) error {
	if err := deleteMessage(params, entry); err != nil {
		if strings.Contains(err.Error(), "message to delete not found") {
			log.Printf("[INFO] delete message, %v", err)
		} else {
			return err
		}
	}
	result := params.DB.Unscoped().Delete(entry)
	if result.Error != nil {
		return result.Error
	}
	time.Sleep(TimeoutBetweenMessages)
	return nil
}

func sendMessage(params *Params, msg tgbotapi.Chattable) error {
	sendedMsg, err := params.Bot.Send(msg)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			log.Printf("[INFO] send message, %v", err)
		} else {
			return err
		}
	}
	var Item = params.Item
	pubDate, err := time.Parse(time.RFC1123Z, Item.Published)
	if err != nil {
		log.Fatalf("[ERROR] parse date, %v", err)
	}

	var entry db.Entry
	result := params.DB.First(&entry, "guid = ? AND updated_at > NOW() - INTERVAL '2 hours'", Item.GUID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = params.DB.Create(&db.Entry{
			GUID:        Item.GUID,
			Provider:    params.Provider,
			Link:        Item.Link,
			Title:       Item.Title,
			Description: Item.Description,
			Categories:  Item.Categories,
			Published:   pubDate,
			MessageID:   sendedMsg.MessageID,
		})
	} else {
		entry.Title = Item.Title
		entry.Description = Item.Description
		entry.Link = Item.Link
		result = params.DB.Where("guid = ?", Item.GUID).Updates(&entry)
	}
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func addMessage(params *Params) error {
	var Item = params.Item
	msg, err := createNewMessageObject(params, &Message{
		FeedTitle:   params.Feed.Title,
		Title:       Item.Title,
		Description: Item.Description,
		Categories:  Item.Categories,
		Link:        Item.Link,
		ImageURL:    getImageURL(Item),
	})
	if err != nil {
		log.Fatalf("[ERROR] get message, %v", err)
	}
	return sendMessage(params, msg)
}

func editMessage(params *Params, entry db.Entry) error {
	var Item = params.Item
	msg := createEditMessageObject(params, entry.MessageID, &Message{
		FeedTitle:   params.Feed.Title,
		Title:       Item.Title,
		Description: Item.Description,
		Link:        Item.Link,
		ImageURL:    getImageURL(Item),
	})
	return sendMessage(params, msg)
}

func deleteMessage(params *Params, entry db.Entry) error {
	msg := createDeleteMessageObject(params, entry.MessageID)
	_, err := params.Bot.Send(msg)
	if err != nil {
		return err
	}
	return nil
}

func getProviders(dbConnect *gorm.DB) []db.Provider {
	var providers []db.Provider
	result := dbConnect.Find(&providers)
	if result.Error != nil {
		log.Fatalf("[ERROR] get providers, %v", result.Error)
	}
	return providers
}

func deleteDeletedEntries(params *Params, entries *[]db.Entry, items *[]*gofeed.Item) error {
	for _, entry := range *entries {
		notFoundEntry := true
		for _, item := range *items {
			if entry.GUID == item.GUID {
				notFoundEntry = false
				break
			}
		}
		if notFoundEntry {
			if err := deleteRecord(params, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

func intersect(arr1, arr2 []string) []string {
	m := map[string]int{}
	var res []string
	for _, n := range arr1 {
		m[n]++
	}
	for _, n := range arr2 {
		if m[n] > 0 {
			res = append(res, n)
			m[n]--
		}
	}
	return res
}

func isValidItem(params *Params, item *gofeed.Item) bool {
	pubDate, _ := time.Parse(time.RFC1123Z, item.Published)
	if len(intersect(params.BlockedCategories, item.Categories)) > 0 {
		return false
	}
	if pubDate.Add(CheckFromTime).Before(time.Now()) {
		return false
	}
	if item.Description == "" {
		return false
	}
	return true
}

func findSimilarRecord(params *Params, item *gofeed.Item) (bool, error) {
	var entry db.Entry
	result := params.DB.First(&entry, "updated_at > NOW() - INTERVAL '1 day' AND provider_id != ? AND similarity(?,title) > 0.3", params.Provider.ID, item.Title)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func addMissingEntries(params *Params, entries *[]db.Entry, items *[]*gofeed.Item) error {
	for _, item := range *items {
		if !isValidItem(params, item) {
			continue
		}

		found, err := findSimilarRecord(params, item)
		if err != nil {
			return err
		}
		if found {
			continue
		}

		log.Printf("[INFO] add/edit record with guid, %v", item.GUID)
		params.Item = item
		if err := addRecord(params, entries); err != nil {
			return err
		}
	}
	return nil
}

func cleanUp(dbConnect *gorm.DB) {
	ticker := time.NewTicker(24 * time.Hour)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			dbConnect.Unscoped().Where("updated_at < NOW() - INTERVAL '1 month'").Delete(&db.Entry{})
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func main() {
	_ = godotenv.Load()
	dbConnect := db.Connect(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	bot, chatID := tg.Connect(os.Getenv("TELEGRAM_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
	bot.Debug = os.Getenv("DEBUG") == "true"
	log.Printf("Authorized on account %s", bot.Self.UserName)

	go cleanUp(dbConnect)

	ticker := time.NewTicker(TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			providers := getProviders(dbConnect)
			for _, provider := range providers {
				if provider.Lang != os.Getenv("LANG_NEWS") {
					continue
				}
				feed, err := http.GetFeed(provider.URL)
				if err != nil {
					log.Fatalf("[ERROR] get feed, %v", err)
				}
				var entries []db.Entry
				result := dbConnect.Where("updated_at > NOW() - INTERVAL '2 hours' AND provider_id = ?", provider.ID).Find(&entries)
				if result.Error != nil {
					log.Fatalf("[ERROR] query entries, %v", result.Error)
				}
				params := &Params{Bot: bot, DB: dbConnect, Feed: feed, Provider: provider, ChatID: chatID, BlockedCategories: provider.BlockedCategories}
				if err := deleteDeletedEntries(params, &entries, &feed.Items); err != nil {
					log.Fatalf("[ERROR] delete record, %v", err)
				}
				if err := addMissingEntries(params, &entries, &feed.Items); err != nil {
					log.Fatalf("[ERROR] add/edit record, %v", err)
				}
			}
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"

	"github.com/better-than-yours/estonia-news/db"
	"github.com/better-than-yours/estonia-news/rest"
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
	Link        string
	ImageURL    string
}

// Params - params
type Params struct {
	Bot      *tgbotapi.BotAPI
	DB       *gorm.DB
	Feed     *gofeed.Feed
	Item     *gofeed.Item
	Provider db.Provider
	ChatID   int64
	Lang     string
}

func getFeed(url string) *gofeed.Feed {
	response, err := rest.Get(url)
	if err != nil {
		log.Fatalf("[ERROR] get feed, %v", err)
	}
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(response))
	if err != nil {
		log.Fatalf("[ERROR] parse feed, %v", err)
	}
	return feed
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

func getText(params *Params, msg *Message) string {
	title := formatText(msg.Title)
	description := formatText(msg.Description)
	if params.Provider.Lang == "EST" {
		if title != "" {
			text, err := rest.Translate(title, "et", "en")
			if err != nil {
				log.Fatalf("[ERROR] get translate, %v (%s)", err, title)
			}
			title = text
		}
		if description != "" {
			text, err := rest.Translate(description, "et", "en")
			if err != nil {
				log.Fatalf("[ERROR] get translate, %v (%s)", err, description)
			}
			description = text
		}
	}
	return fmt.Sprintf("<b>%s</b>\n\n%s\n\n<a href=\"%s\">%s</a>", title, description, msg.Link, strings.TrimSpace(msg.FeedTitle))
}

func getImage(imageURL string) ([]byte, error) {
	response, err := http.Get(imageURL) // #nosec G107
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func createNewMessageObject(params *Params, msg *Message) (*tgbotapi.PhotoConfig, error) {
	content, err := getImage(msg.ImageURL)
	if err != nil {
		return nil, err
	}
	file := tgbotapi.FileBytes{Name: msg.ImageURL, Bytes: content}
	text := getText(params, msg)
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
	text := getText(params, msg)
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
	result := params.DB.First(&entry, "guid = ?", Item.GUID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = params.DB.Create(&db.Entry{GUID: Item.GUID, Provider: params.Provider, Link: Item.Link, Title: Item.Title, Description: Item.Description, Published: pubDate, MessageID: sendedMsg.MessageID})
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

func isValidItem(item *gofeed.Item) bool {
	pubDate, _ := time.Parse(time.RFC1123Z, item.Published)
	if pubDate.Add(CheckFromTime).Before(time.Now()) {
		return false
	}
	if item.Description == "" {
		return false
	}
	return true
}

func addMissingEntries(params *Params, entries *[]db.Entry, items *[]*gofeed.Item) error {
	for _, item := range *items {
		if !isValidItem(item) {
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
			dbConnect.Unscoped().Where("published < NOW() - INTERVAL '1 month'").Delete(&db.Entry{})
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
	providers := getProviders(dbConnect)

	go cleanUp(dbConnect)

	ticker := time.NewTicker(TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			for _, provider := range providers {
				if provider.Lang != os.Getenv("LANG_NEWS") {
					continue
				}
				feed := getFeed(provider.URL)
				var entries []db.Entry
				result := dbConnect.Where("published > NOW() - INTERVAL '6 hours' AND provider_id = ?", provider.ID).Find(&entries)
				if result.Error != nil {
					log.Fatalf("[ERROR] query entries, %v", result.Error)
				}
				params := &Params{Bot: bot, DB: dbConnect, Feed: feed, Provider: provider, ChatID: chatID}
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

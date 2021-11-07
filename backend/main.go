package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"gorm.io/gorm"

	"github.com/better-than-yours/estonia-news/db"
	"github.com/better-than-yours/estonia-news/http"
	"github.com/better-than-yours/estonia-news/tg"
)

// TimeoutBetweenLoops - Main loop timeout
var TimeoutBetweenLoops = 5 * time.Minute

// TimeoutBetweenMessages - timeout between attempts to send a message
var TimeoutBetweenMessages = 5 * time.Second

// TimeShift is time shift
var TimeShift = 1

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
	BlockedWords      []string
}

func formatGUID(url string) string {
	var r *regexp.Regexp
	r = regexp.MustCompile(`^\w+#\d+$`)
	if r.MatchString(url) {
		return url
	}
	r = regexp.MustCompile(`err.*?/(\d+)$`)
	if r.MatchString(url) {
		return fmt.Sprintf("err#%s", r.FindStringSubmatch(url)[1])
	}
	r = regexp.MustCompile(`delfi.*?/(\d+)/.*?$`)
	if r.MatchString(url) {
		return fmt.Sprintf("delfi#%s", r.FindStringSubmatch(url)[1])
	}
	return ""
}

func formatText(text string) string {
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`<img.*?/>`).ReplaceAllString(text, "")
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
				log.Fatalf("[ERROR] get translate '%s', %v", title, err)
			}
			title = text
		}
		if description != "" {
			text, err := http.Translate(description, "et", "en")
			if err != nil {
				log.Fatalf("[ERROR] get translate '%s', %v", description, err)
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

func addRecord(params *Params) error {
	var Item = params.Item
	var entry db.Entry
	result := params.DB.First(&entry, "guid = ?", Item.GUID)
	var err error
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if hasChanges(Item, entry) {
			log.Printf("[INFO] send edit message '%s'", formatGUID(entry.GUID))
			err = editMessage(params, entry)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if !isValidItemByTerm(Item) {
		return nil
	}
	log.Printf("[INFO] send message '%s'", Item.GUID)
	err = addMessage(params)
	if err != nil {
		return err
	}
	time.Sleep(TimeoutBetweenMessages)
	return nil
}

func deleteRecord(params *Params, entry db.Entry) error {
	if err := deleteMessage(params, entry); err != nil {
		if strings.Contains(err.Error(), "message to delete not found") {
			log.Printf("[INFO] delete message '%s', %v", formatGUID(entry.GUID), err)
		} else {
			return err
		}
	}
	result := params.DB.Unscoped().Where("guid = ?", formatGUID(entry.GUID)).Select("EntryToCategory").Delete(&db.Entry{})
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
		entry = db.Entry{
			GUID:        Item.GUID,
			Provider:    params.Provider,
			Link:        Item.Link,
			Title:       Item.Title,
			Description: Item.Description,
			Published:   pubDate,
			MessageID:   sendedMsg.MessageID,
		}
		result = params.DB.Create(&entry)
		if result.Error != nil {
			return result.Error
		}
		var entryToCategory []db.EntryToCategory
		for _, categoryName := range Item.Categories {
			category := db.Category{
				Name:     categoryName,
				Provider: params.Provider,
			}
			result = db.UpsertCategory(params.DB, &category)
			if result.Error != nil {
				return result.Error
			}
			entryToCategory = append(entryToCategory, db.EntryToCategory{
				Entry:    entry,
				Category: category,
			})
		}
		result = params.DB.Create(&entryToCategory)
		if result.Error != nil {
			return result.Error
		}
	} else {
		entry.Title = Item.Title
		entry.Description = Item.Description
		entry.Link = Item.Link
		result = params.DB.Where("guid = ?", Item.GUID).Updates(&entry)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func addMessage(params *Params) error {
	var Item = params.Item
	imageURL, err := http.GetImageURL(Item.Link)
	if err != nil {
		log.Fatalf("[ERROR] get image url, %v", err)
	}
	msg, err := createNewMessageObject(params, &Message{
		FeedTitle:   params.Feed.Title,
		Title:       Item.Title,
		Description: Item.Description,
		Categories:  Item.Categories,
		Link:        Item.Link,
		ImageURL:    imageURL,
	})
	if err != nil {
		log.Fatalf("[ERROR] get message, %v", err)
	}
	return sendMessage(params, msg)
}

func editMessage(params *Params, entry db.Entry) error {
	var Item = params.Item
	imageURL, err := http.GetImageURL(Item.Link)
	if err != nil {
		log.Fatalf("[ERROR] get image url, %v", err)
	}
	msg := createEditMessageObject(params, entry.MessageID, &Message{
		FeedTitle:   params.Feed.Title,
		Title:       Item.Title,
		Description: Item.Description,
		Link:        Item.Link,
		ImageURL:    imageURL,
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

func deleteDeletedEntries(params *Params, items []*gofeed.Item) error {
	var entries []db.Entry
	result := params.DB.Where(fmt.Sprintf("published > NOW() - INTERVAL '%d hours' AND provider_id = %d", TimeShift, params.Provider.ID)).Find(&entries)
	if result.Error != nil {
		log.Fatalf("[ERROR] query entries, %v", result.Error)
	}
	items = funk.Filter(items, isValidItemByTerm).([]*gofeed.Item)
	for _, entry := range entries {
		foundEntry := funk.Contains(items, func(item *gofeed.Item) bool {
			return formatGUID(entry.GUID) == item.GUID
		})
		if !foundEntry {
			if err := deleteRecord(params, entry); err != nil {
				log.Printf("[INFO] delete record '%s', %v", formatGUID(entry.GUID), err)
				return err
			}
		}
	}
	return nil
}

func isValidItemByContent(params *Params, item *gofeed.Item) bool {
	if len(funk.IntersectString(params.BlockedCategories, item.Categories)) > 0 {
		return false
	}
	foundBlockedWords := funk.FilterString(params.BlockedWords, func(word string) bool {
		return strings.Contains(item.Title, word) || strings.Contains(item.Description, word)
	})
	if len(foundBlockedWords) > 0 {
		return false
	}
	if item.Description == "" {
		return false
	}
	return true
}

func isValidItemByTerm(item *gofeed.Item) bool {
	pubDate, _ := time.Parse(time.RFC1123Z, item.Published)
	return !pubDate.Add(time.Duration(TimeShift) * time.Hour).Before(time.Now())
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

func addMissingEntries(params *Params, items []*gofeed.Item) error {
	for _, item := range items {
		found, err := findSimilarRecord(params, item)
		if err != nil {
			log.Printf("[INFO] find similar record '%s', %v", item.GUID, err)
			return err
		}
		if found {
			continue
		}
		params.Item = item
		if err := addRecord(params); err != nil {
			log.Printf("[INFO] add record '%s', %v", item.GUID, err)
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
			dbConnect.Unscoped().Where("updated_at < NOW() - INTERVAL '7 days'").Select("EntryToCategory").Delete(&db.Entry{})
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
	job(dbConnect, bot, chatID)

	ticker := time.NewTicker(TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			job(dbConnect, bot, chatID)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func job(dbConnect *gorm.DB, bot *tgbotapi.BotAPI, chatID int64) {
	providers := getProviders(dbConnect)
	for _, provider := range providers {
		if provider.Lang != os.Getenv("LANG_NEWS") {
			continue
		}
		feed, err := http.GetFeed(provider.URL)
		if err != nil {
			log.Fatalf("[ERROR] get feed, %v", err)
		}
		params := &Params{
			Bot:               bot,
			DB:                dbConnect,
			Feed:              feed,
			Provider:          provider,
			ChatID:            chatID,
			BlockedCategories: provider.BlockedCategories,
			BlockedWords:      provider.BlockedWords,
		}
		feed.Items = funk.Map(feed.Items, func(item *gofeed.Item) *gofeed.Item {
			if len(item.GUID) > 0 {
				item.GUID = formatGUID(item.GUID)
				return item
			}
			link := item.Extensions["feedburner"]["origLink"][0].Value
			return &gofeed.Item{
				GUID:        formatGUID(link),
				Link:        link,
				Title:       item.Title,
				Description: item.Description,
				Categories:  item.Categories,
				Published:   item.Published,
			}
		}).([]*gofeed.Item)
		feed.Items = funk.Filter(feed.Items, func(item *gofeed.Item) bool {
			return isValidItemByContent(params, item)
		}).([]*gofeed.Item)
		sort.Slice(feed.Items, func(i, j int) bool {
			return feed.Items[i].Published > feed.Items[j].Published
		})
		if err := deleteDeletedEntries(params, feed.Items); err != nil {
			log.Fatalf("[ERROR] delete record")
		}
		if err := addMissingEntries(params, feed.Items); err != nil {
			log.Fatalf("[ERROR] add/edit record")
		}
	}
}

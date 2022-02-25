package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thoas/go-funk"
	"gorm.io/gorm"
)

// TimeoutBetweenLoops - Main loop timeout
var TimeoutBetweenLoops = 5 * time.Minute

// TimeoutBetweenMessages - timeout between attempts to send a message
var TimeoutBetweenMessages = 1 * time.Second

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
	Provider          Provider
	ChatID            int64
	Lang              string
	BlockedCategories []string
	BlockedWords      []string
}

func formatGUID(path string) string {
	var r *regexp.Regexp
	r = regexp.MustCompile(`^\w+#\d+$`)
	if r.MatchString(path) {
		return path
	}
	r = regexp.MustCompile(`err.*?/(\d+)$`)
	if r.MatchString(path) {
		return fmt.Sprintf("err#%s", r.FindStringSubmatch(path)[1])
	}
	r = regexp.MustCompile(`delfi.*?/(\d+)/.*?$`)
	if r.MatchString(path) {
		return fmt.Sprintf("delfi#%s", r.FindStringSubmatch(path)[1])
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
			text, err := Translate(title, "et", "en")
			if err != nil {
				taskErrors.With(prometheus.Labels{"error": "get_translate"}).Inc()
				pushMetrics()
				l.Logf("FATAL get translate '%s', %v", title, err)
			}
			title = text
		}
		if description != "" {
			text, err := Translate(description, "et", "en")
			if err != nil {
				taskErrors.With(prometheus.Labels{"error": "get_translate"}).Inc()
				pushMetrics()
				l.Logf("FATAL get translate '%s', %v", description, err)
			}
			description = text
		}
	}
	return
}

func renderMessageBlock(msg *Message, title, description string) string {
	return fmt.Sprintf("<b>%s</b>\n\n%s\n\n<a href=%q>%s</a>", title, description, msg.Link, strings.TrimSpace(msg.FeedTitle))
}

func createNewMessageObject(params *Params, msg *Message) (tgbotapi.Chattable, error) {
	title, description := getText(params, msg)
	text := renderMessageBlock(msg, title, description)

	var config tgbotapi.Chattable
	if msg.ImageURL == "" {
		config = &tgbotapi.MessageConfig{
			BaseChat:              tgbotapi.BaseChat{ChatID: params.ChatID},
			Text:                  text,
			ParseMode:             tgbotapi.ModeHTML,
			DisableWebPagePreview: true,
		}
	} else {
		content, err := GetImage(msg.ImageURL)
		if err != nil {
			return nil, err
		}
		file := tgbotapi.FileBytes{Name: msg.ImageURL, Bytes: content}
		config = &tgbotapi.PhotoConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat:    tgbotapi.BaseChat{ChatID: params.ChatID},
				File:        file,
				UseExisting: false,
			},
			Caption:   text,
			ParseMode: tgbotapi.ModeHTML,
		}

	}
	return config, nil
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

func hasChanges(item *gofeed.Item, entry Entry) bool {
	if entry.Title != item.Title || entry.Description != item.Description || entry.Link != item.Link {
		return true
	}
	return false
}

func addRecord(params *Params) error {
	var Item = params.Item
	var entry Entry
	result := params.DB.First(&entry, "guid = ?", Item.GUID)
	var err error
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if hasChanges(Item, entry) {
			l.Logf("INFO send edit message '%s'", formatGUID(entry.GUID))
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
	l.Logf("INFO send message '%s'", Item.GUID)
	err = addMessage(params)
	if err != nil {
		return err
	}
	time.Sleep(TimeoutBetweenMessages)
	return nil
}

func deleteRecord(params *Params, entry Entry) error {
	if err := deleteMessage(params, entry); err != nil {
		if strings.Contains(err.Error(), "message to delete not found") {
			l.Logf("ERROR delete message '%s', %v", formatGUID(entry.GUID), err)
			taskErrors.With(prometheus.Labels{"error": "delete_message"}).Inc()
		} else {
			return err
		}
	}
	// TODO need to fix it
	result := params.DB.Unscoped().Where("entry_id = ?", formatGUID(entry.GUID)).Delete(&EntryToCategory{})
	if result.Error != nil {
		return result.Error
	}
	result = params.DB.Unscoped().Where("guid = ?", formatGUID(entry.GUID)).Delete(&Entry{})
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
			l.Logf("ERROR send message, %v", err)
			taskErrors.With(prometheus.Labels{"error": "send_message"}).Inc()
		} else if strings.Contains(err.Error(), "there is no caption in the message to edit") {
			l.Logf("ERROR send message, %v", err)
			taskErrors.With(prometheus.Labels{"error": "send_message"}).Inc()
		} else {
			return err
		}
	}
	var Item = params.Item
	pubDate, err := time.Parse(time.RFC1123Z, Item.Published)
	if err != nil {
		taskErrors.With(prometheus.Labels{"error": "parse_date"}).Inc()
		pushMetrics()
		l.Logf("FATAL parse date, %v", err)
	}

	var entry Entry
	result := params.DB.First(&entry, "guid = ?", Item.GUID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		entry = Entry{
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
		var entryToCategory []EntryToCategory
		for _, categoryName := range Item.Categories {
			category := Category{
				Name:     categoryName,
				Provider: params.Provider,
			}
			result = UpsertCategory(params.DB, &category)
			if result.Error != nil {
				return result.Error
			}
			entryToCategory = append(entryToCategory, EntryToCategory{
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
	imageURL, err := GetImageURL(Item.Link)
	if err != nil {
		l.Logf("ERROR get image url, %v", err)
		taskErrors.With(prometheus.Labels{"error": "get_image_url"}).Inc()
		pushMetrics()
		imageURL = ""
	}
	_, err = url.ParseRequestURI(imageURL)
	if err != nil {
		l.Logf("ERROR parse image url, %v", err)
		taskErrors.With(prometheus.Labels{"error": "parse_image_url"}).Inc()
		imageURL = ""
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
		taskErrors.With(prometheus.Labels{"error": "get_message"}).Inc()
		pushMetrics()
		l.Logf("FATAL get message, %v", err)
	}
	return sendMessage(params, msg)
}

func editMessage(params *Params, entry Entry) error {
	var Item = params.Item
	imageURL, err := GetImageURL(Item.Link)
	if err != nil {
		l.Logf("ERROR get image url, %v", err)
		taskErrors.With(prometheus.Labels{"error": "get_image_url"}).Inc()
		pushMetrics()
		imageURL = ""
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

func deleteMessage(params *Params, entry Entry) error {
	msg := createDeleteMessageObject(params, entry.MessageID)
	_, err := params.Bot.Send(msg)
	if err != nil {
		return err
	}
	return nil
}

func getProviders(dbConnect *gorm.DB) []Provider {
	var providers []Provider
	result := dbConnect.Find(&providers)
	if result.Error != nil {
		taskErrors.With(prometheus.Labels{"error": "get_providers"}).Inc()
		pushMetrics()
		l.Logf("FATAL get providers, %v", result.Error)
	}
	return providers
}

func deleteDeletedEntries(params *Params, items []*gofeed.Item) error {
	var entries []Entry
	result := params.DB.Where(fmt.Sprintf("published > NOW() - INTERVAL '%d hours' AND provider_id = %d", TimeShift, params.Provider.ID)).Find(&entries)
	if result.Error != nil {
		taskErrors.With(prometheus.Labels{"error": "query_entries"}).Inc()
		pushMetrics()
		l.Logf("FATAL query entries, %v", result.Error)
	}
	items = funk.Filter(items, isValidItemByTerm).([]*gofeed.Item)
	for _, entry := range entries {
		foundEntry := funk.Contains(items, func(item *gofeed.Item) bool {
			return formatGUID(entry.GUID) == item.GUID
		})
		if !foundEntry {
			if err := deleteRecord(params, entry); err != nil {
				l.Logf("ERROR delete record '%s', %v", formatGUID(entry.GUID), err)
				taskErrors.With(prometheus.Labels{"error": "delete_record"}).Inc()
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
	var entry Entry
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
			l.Logf("ERROR find similar record '%s', %v", item.GUID, err)
			taskErrors.With(prometheus.Labels{"error": "find_similar_record"}).Inc()
			return err
		}
		if found {
			continue
		}
		params.Item = item
		if err := addRecord(params); err != nil {
			l.Logf("ERROR add record '%s', %v", item.GUID, err)
			taskErrors.With(prometheus.Labels{"error": "add_record"}).Inc()
			return err
		}
	}
	return nil
}

func cleanUp(dbConnect *gorm.DB) {
	ticker := time.NewTicker(12 * time.Hour)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			// TODO need to fix it
			dbConnect.Unscoped().Select("Entry").Where("entries.updated_at < NOW() - INTERVAL '7 days'").Delete(&EntryToCategory{})
			dbConnect.Unscoped().Where("updated_at < NOW() - INTERVAL '7 days'").Delete(&Entry{})
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func main() {
	_ = godotenv.Load()
	metrics(os.Getenv("PROMETHEUS_URL"), os.Getenv("PROMETHEUS_JOB"))
	dbConnect := connectDB(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	bot, chatID := connectTg(os.Getenv("TELEGRAM_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
	bot.Debug = os.Getenv("DEBUG") == "true"
	l.Logf("INFO authorized on account '%s'", bot.Self.UserName)

	go cleanUp(dbConnect)
	job(dbConnect, bot, chatID)
	pushMetrics()

	ticker := time.NewTicker(TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			job(dbConnect, bot, chatID)
			pushMetrics()
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
		feed, err := GetFeed(provider.URL)
		if err != nil {
			taskErrors.With(prometheus.Labels{"error": "get_feed"}).Inc()
			pushMetrics()
			l.Logf("FATAL get feed, %v", err)
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
			return &gofeed.Item{
				GUID:        formatGUID(item.Link),
				Link:        item.Link,
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
			taskErrors.With(prometheus.Labels{"error": "delete_record"}).Inc()
			pushMetrics()
			l.Logf("FATAL delete record, %v", err)
		}
		if err := addMissingEntries(params, feed.Items); err != nil {
			taskErrors.With(prometheus.Labels{"error": "add_edit_record"}).Inc()
			pushMetrics()
			l.Logf("FATAL add/edit record, %v", err)
		}
	}
}

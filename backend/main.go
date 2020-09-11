package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/better-than-yours/estonia-news/model"
	"github.com/better-than-yours/estonia-news/rest"
)

// TimeoutBetweenLoops - Main loop timeout
var TimeoutBetweenLoops = 5 * time.Minute

// TimeoutBetweenMessages - timeout between attempts to send a message
var TimeoutBetweenMessages = 5 * time.Second

// CheckFromTime - time ago from which should check messages
var CheckFromTime = 30 * time.Minute

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
	Provider model.Provider
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
		text, err := rest.Translate(title, "et", "en")
		if err != nil {
			log.Fatalf("[ERROR] get translate, %v (%s)", err, title)
		}
		title = text
		text, err = rest.Translate(description, "et", "en")
		if err != nil {
			log.Fatalf("[ERROR] get translate, %v (%s)", err, description)
		}
		description = text
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

func hasChanges(item *gofeed.Item, entry model.Entry) bool {
	if entry.Title != item.Title || entry.Link != item.Link || entry.Description != item.Description {
		return true
	}
	return false
}

func send(params *Params) error {
	var entries []model.Entry
	result := params.DB.Where("published > NOW() - INTERVAL '24 hours'").Find(&entries)
	if result.Error != nil {
		return result.Error
	}
	var Item = params.Item
	for _, entry := range entries {
		if entry.GUID == Item.GUID {
			if hasChanges(Item, entry) {
				log.Printf("[INFO] send edit item, %v", Item.GUID)
				if editMessageErr := editMessage(params, entry); editMessageErr != nil {
					return editMessageErr
				}
			}
			return nil
		}
	}
	log.Printf("[INFO] send new item, %v", Item.GUID)
	err := addMessage(params)
	if err != nil {
		return err
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
	result := params.DB.Create(&model.Entry{GUID: Item.GUID, Provider: params.Provider, Link: Item.Link, Title: Item.Title, Description: Item.Description, Published: pubDate, MessageID: sendedMsg.MessageID})
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

func editMessage(params *Params, entry model.Entry) error {
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

func getProviders(db *gorm.DB) []model.Provider {
	var providers []model.Provider
	result := db.Find(&providers)
	if result.Error != nil {
		log.Fatalf("[ERROR] get providers, %v", result.Error)
	}
	return providers
}

func connectToTelegram(telegramToken, telegramChatID string) (bot *tgbotapi.BotAPI, chatID int64) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatalf("[ERROR] telegram api, %v", err)
	}
	chatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		log.Fatalf("[ERROR] chat id, %v", err)
	}
	return
}

func connectToDB(dbHost, dbUser, dbPassword, dbName string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s", dbHost, dbUser, dbPassword, dbName)), &gorm.Config{})
	if err != nil {
		log.Fatalf("[ERROR] failed to connect database, %v", err)
	}
	err = db.AutoMigrate(&model.Entry{}, &model.Provider{})
	if err != nil {
		log.Fatalf("[ERROR] db migration, %v", err)
	}
	return db
}

func main() {
	_ = godotenv.Load()
	db := connectToDB(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	bot, chatID := connectToTelegram(os.Getenv("TELEGRAM_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
	bot.Debug = os.Getenv("DEBUG") == "true"
	log.Printf("Authorized on account %s", bot.Self.UserName)

	providers := getProviders(db)

	ticker := time.NewTicker(TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			for _, provider := range providers {
				fmt.Println(os.Getenv("LANG_NEWS"))
				if provider.Lang != os.Getenv("LANG_NEWS") {
					continue
				}
				feed := getFeed(provider.URL)
				for _, item := range feed.Items {
					pubDate, err := time.Parse(time.RFC1123Z, item.Published)
					if err != nil {
						log.Fatalf("[ERROR] parse date, %v", err)
					}
					if pubDate.Add(CheckFromTime).Before(time.Now()) {
						continue
					}
					log.Printf("[INFO] send item, %v", item.GUID)
					if err := send(&Params{Bot: bot, DB: db, Feed: feed, Item: item, Provider: provider, ChatID: chatID}); err != nil {
						log.Fatalf("[ERROR] send, %v", err)
					}
				}
			}
		case <-quit:
			ticker.Stop()
			log.Println("stop")
			return
		}
	}
}

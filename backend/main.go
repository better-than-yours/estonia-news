package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/mmcdole/gofeed"

	"github.com/lafin/estonia-news/model"
	"github.com/lafin/estonia-news/proc"
	"github.com/lafin/estonia-news/rest"
)

// ChatID - Telegram chat ID
var ChatID = os.Getenv("TELEGRAM_CHAT_ID")

// Token - Telegram token
var Token = os.Getenv("TELEGRAM_TOKEN")

// TimeoutBetweenLoops - Main loop timeout
var TimeoutBetweenLoops = 5 * time.Minute

// TimeoutBetweenMessages - timeout between attempts to send a message
var TimeoutBetweenMessages = 5 * time.Second

// MessageConfig - config
type MessageConfig struct {
	FeedTitle   string
	Title       string
	Description string
	Link        string
	ImageURL    string
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

func getText(msg *MessageConfig) string {
	return fmt.Sprintf("<b>%s</b>\n\n%s\n\n<a href=\"%s\">%s</a>", msg.Title, msg.Description, msg.Link, msg.FeedTitle)
}

func getImage(imageURL string) ([]byte, error) {
	response, err := http.Get(imageURL) // #nosec G107
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func createNewMessageObject(msg *MessageConfig) (*tgbotapi.PhotoConfig, error) {
	chatID, err := strconv.ParseInt(ChatID, 10, 64)
	if err != nil {
		return nil, err
	}
	content, err := getImage(msg.ImageURL)
	if err != nil {
		return nil, err
	}
	file := tgbotapi.FileBytes{Name: msg.ImageURL, Bytes: content}
	return &tgbotapi.PhotoConfig{
		BaseFile: tgbotapi.BaseFile{
			BaseChat:    tgbotapi.BaseChat{ChatID: chatID},
			File:        file,
			UseExisting: false,
		},
		Caption:   getText(msg),
		ParseMode: tgbotapi.ModeHTML,
	}, nil
}

func createEditMessageObject(messageID int, msg *MessageConfig) (*tgbotapi.EditMessageCaptionConfig, error) {
	chatID, err := strconv.ParseInt(ChatID, 10, 64)
	if err != nil {
		return nil, err
	}
	return &tgbotapi.EditMessageCaptionConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:    chatID,
			MessageID: messageID,
		},
		Caption:   getText(msg),
		ParseMode: tgbotapi.ModeHTML,
	}, nil
}

func hasChanges(item *gofeed.Item, entry model.Entry) bool {
	if entry.Title != item.Title || entry.Link != item.Link || entry.Description != item.Description {
		return true
	}
	return false
}

func send(bot *tgbotapi.BotAPI, db *proc.Processor, feed *gofeed.Feed, item *gofeed.Item) error {
	entries, err := db.LoadEntry()
	if err != nil {
		return err
	}
	for _, entry := range *entries {
		if entry.GUID == item.GUID {
			if hasChanges(item, entry) {
				log.Printf("[INFO] send edit item, %v", item.GUID)
				if editMessageErr := editMessage(bot, db, feed, item, entry); editMessageErr != nil {
					return editMessageErr
				}
			}
			return nil
		}
	}
	log.Printf("[INFO] send new item, %v", item.GUID)
	err = addMessage(bot, db, feed, item)
	if err != nil {
		return err
	}
	time.Sleep(TimeoutBetweenMessages)
	return nil
}

func sendMessage(bot *tgbotapi.BotAPI, db *proc.Processor, item *gofeed.Item, msg tgbotapi.Chattable) error {
	sendedMsg, err := bot.Send(msg)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			log.Printf("[INFO] send message, %v", err)
		} else {
			return err
		}
	}
	pubDate, err := time.Parse(time.RFC1123Z, item.Published)
	if err != nil {
		log.Fatalf("[ERROR] parse date, %v", err)
	}
	if _, err := db.SaveEntry(model.Entry{GUID: item.GUID, Link: item.Link, Title: item.Title, Description: item.Description, Published: pubDate, MessageID: sendedMsg.MessageID}); err != nil {
		return err
	}
	return nil
}

func addMessage(bot *tgbotapi.BotAPI, db *proc.Processor, feed *gofeed.Feed, item *gofeed.Item) error {
	msg, err := createNewMessageObject(&MessageConfig{
		FeedTitle:   feed.Title,
		Title:       item.Title,
		Description: item.Description,
		Link:        item.Link,
		ImageURL:    getImageURL(item),
	})
	if err != nil {
		log.Fatalf("[ERROR] get message, %v", err)
	}
	return sendMessage(bot, db, item, msg)
}

func editMessage(bot *tgbotapi.BotAPI, db *proc.Processor, feed *gofeed.Feed, item *gofeed.Item, entry model.Entry) error {
	msg, err := createEditMessageObject(entry.MessageID, &MessageConfig{
		FeedTitle:   feed.Title,
		Title:       item.Title,
		Description: item.Description,
		Link:        item.Link,
		ImageURL:    getImageURL(item),
	})
	if err != nil {
		log.Fatalf("[ERROR] get message, %v", err)
	}
	return sendMessage(bot, db, item, msg)
}

func main() {
	providers := []struct {
		URL string
	}{
		{
			URL: "https://news.postimees.ee/rss",
		},
		{
			URL: "https://news.err.ee/rss",
		},
	}
	db, err := proc.NewBoltDB("var/store.bdb")
	if err != nil {
		log.Fatalf("[ERROR] can't open db, %v", err)
	}
	bot, err := tgbotapi.NewBotAPI(Token)
	if err != nil {
		log.Fatalf("[ERROR] telegram api, %v", err)
	}
	bot.Debug = os.Getenv("DEBUG") == "true"
	log.Printf("Authorized on account %s", bot.Self.UserName)

	ticker := time.NewTicker(TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			for _, provider := range providers {
				feed := getFeed(provider.URL)
				for _, item := range feed.Items {
					pubDate, err := time.Parse(time.RFC1123Z, item.Published)
					if err != nil {
						log.Fatalf("[ERROR] parse date, %v", err)
					}
					if pubDate.Add(10 * time.Hour).Before(time.Now()) {
						continue
					}
					log.Printf("[INFO] send item, %v", item.GUID)
					if err := send(bot, db, feed, item); err != nil {
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

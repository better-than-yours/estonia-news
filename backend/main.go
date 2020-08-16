package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/mmcdole/gofeed"

	"github.com/lafin/estonia-news/model"
	"github.com/lafin/estonia-news/proc"
	"github.com/lafin/estonia-news/rest"
)

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
		log.Panic(err)
	}
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(response))
	if err != nil {
		log.Panic(err)
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

func getMessage(channel string, msg *MessageConfig) (*tgbotapi.PhotoConfig, error) {
	content, err := getImage(msg.ImageURL)
	if err != nil {
		return nil, err
	}
	file := tgbotapi.FileBytes{Name: msg.ImageURL, Bytes: content}
	return &tgbotapi.PhotoConfig{
		BaseFile: tgbotapi.BaseFile{
			BaseChat:    tgbotapi.BaseChat{ChannelUsername: channel},
			File:        file,
			UseExisting: false,
		},
		Caption:   getText(msg),
		ParseMode: tgbotapi.ModeHTML,
	}, nil
}

func sendMessage(db *proc.Processor, item *gofeed.Item, bot *tgbotapi.BotAPI, msg *tgbotapi.PhotoConfig) error {
	found := false
	entries, err := db.LoadEntry()
	if err != nil {
		return err
	}
	for _, entry := range *entries {
		if entry.GUID == item.GUID {
			found = true
		}
	}
	if !found {
		sendedMsg, err := bot.Send(msg)
		if err != nil {
			return err
		}
		pubDate, err := time.Parse(time.RFC1123Z, item.Published)
		if err != nil {
			log.Panic(err)
		}
		if _, err := db.SaveEntry(model.Entry{GUID: item.GUID, Link: item.Link, Title: item.Title, Published: pubDate, MessageID: sendedMsg.MessageID}); err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func main() {
	channelUsername := "@ee_news"
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
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = os.Getenv("DEBUG") == "true"
	log.Printf("Authorized on account %s", bot.Self.UserName)

	ticker := time.NewTicker(5 * time.Minute)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			for _, provider := range providers {
				feed := getFeed(provider.URL)
				for _, item := range feed.Items {
					pubDate, err := time.Parse(time.RFC1123Z, item.Published)
					if err != nil {
						log.Panic(err)
					}
					if pubDate.Add(10 * time.Hour).Before(time.Now()) {
						continue
					}
					msg, err := getMessage(channelUsername, &MessageConfig{
						FeedTitle:   feed.Title,
						Title:       item.Title,
						Description: item.Description,
						Link:        item.Link,
						ImageURL:    getImageURL(item),
					})
					if err != nil {
						log.Panic(err)
					}
					if err := sendMessage(db, item, bot, msg); err != nil {
						log.Panic(err)
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

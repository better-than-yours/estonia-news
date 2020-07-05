package main

import (
	"fmt"
	"log"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/mmcdole/gofeed"

	"github.com/lafin/estonia-news/model"
	"github.com/lafin/estonia-news/proc"
	"github.com/lafin/estonia-news/rest"
)

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

func getMessage(item *gofeed.Item, feed *gofeed.Feed) *tgbotapi.MessageConfig {
	if len(item.Enclosures) > 0 && item.Enclosures[0].URL != "" {
		item.Image = &gofeed.Image{
			URL: item.Enclosures[0].URL,
		}
	} else if len(item.Extensions["media"]["thumbnail"]) > 0 && item.Extensions["media"]["thumbnail"][0].Attrs["url"] != "" {
		item.Image = &gofeed.Image{
			URL: item.Extensions["media"]["thumbnail"][0].Attrs["url"],
		}
	}
	msg := tgbotapi.NewMessageToChannel("@ee_news", fmt.Sprintf("<b>%s</b>\n\n%s<a href=\"%s\">&#160;</a>\n\n<a href=\"%s\">%s</a>", item.Title, item.Description, item.Image.URL, item.Link, feed.Title))
	msg.ParseMode = tgbotapi.ModeHTML
	return &msg
}

func sendMessage(db *proc.Processor, item *gofeed.Item, bot *tgbotapi.BotAPI, msg *tgbotapi.MessageConfig) error {
	found := false
	entries, err := db.LoadEntry()
	if err != nil {
		return err
	}
	for _, entry := range *entries {
		if entry.Link == item.Link {
			found = true
		}
	}
	if !found {
		if _, err := bot.Send(msg); err != nil {
			return err
		}
		if _, err := db.SaveEntry(model.Entry{Link: item.Link, Title: item.Title}); err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
	return nil
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
					msg := getMessage(item, feed)
					pubDate, err := time.Parse(time.RFC1123Z, item.Published)
					if err != nil {
						log.Panic(err)
					}
					if pubDate.Add(10 * time.Hour).Before(time.Now()) {
						continue
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

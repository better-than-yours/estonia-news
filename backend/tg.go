package main

import (
	"net/url"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/prometheus/client_golang/prometheus"
)

func connectTg(telegramToken, telegramChatID string) (bot *tgbotapi.BotAPI, chatID int64) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		taskErrors.With(prometheus.Labels{"error": "tg_api"}).Inc()
		pushMetrics()
		l.Logf("FATAL telegram api, %v", err)
	}
	chatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		taskErrors.With(prometheus.Labels{"error": "tg_chat_id"}).Inc()
		pushMetrics()
		l.Logf("FATAL chat id, %v", err)
	}
	return
}

// Message - config
type Message struct {
	FeedTitle   string
	Title       string
	Description string
	Categories  []string
	Link        string
	ImageURL    string
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

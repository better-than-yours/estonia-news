package service

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func renderMessageBlock(msg *Message, title, description string) string {
	return fmt.Sprintf("<b>%s</b>\n\n%s\n\n<a href=%q>%s</a>", title, description, msg.Link, strings.TrimSpace(msg.FeedTitle))
}

// FormatText return formated test
func FormatText(text string) string {
	text = regexp.MustCompile(`<img.*?/>`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`([\x{0020}\x{00a0}\x{1680}\x{180e}\x{2000}-\x{200b}\x{202f}\x{205f}\x{3000}\x{feff}])`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n+\s+`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`window.addEventListener\(.*?(false|true)\);`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(#[\w-]+\s+|\.[\w-]+\s+)+{.*?}`).ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	return text
}

func getText(params *config.Params, msg *Message) (title, description string) {
	title = FormatText(msg.Title)
	description = FormatText(msg.Description)
	if params.Provider.Lang == "EST" {
		if title != "" {
			text, err := translate(title, "et", "en")
			if err != nil {
				misc.Fatal("get_translate", "get translate", err)
			}
			title = text
		}
		if description != "" {
			text, err := translate(description, "et", "en")
			if err != nil {
				misc.Fatal("get_translate", "get translate", err)
			}
			description = text
		}
	}
	return
}

func createMessageObject(params *config.Params, msg *Message) (tgbotapi.Chattable, error) {
	title, description := getText(params, msg)
	text := renderMessageBlock(msg, title, description)

	var obj tgbotapi.Chattable
	if msg.ImageURL == "" {
		obj = tgbotapi.MessageConfig{
			BaseChat:              tgbotapi.BaseChat{ChatID: params.ChatID},
			Text:                  text,
			ParseMode:             tgbotapi.ModeHTML,
			DisableWebPagePreview: true,
		}
	} else {
		content, err := getImage(msg.ImageURL)
		if err != nil {
			return nil, err
		}
		file := tgbotapi.FileBytes{Name: msg.ImageURL, Bytes: content}
		obj = tgbotapi.PhotoConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat:    tgbotapi.BaseChat{ChatID: params.ChatID},
				File:        file,
				UseExisting: false,
			},
			Caption:   text,
			ParseMode: tgbotapi.ModeHTML,
		}

	}
	return obj, nil
}

func editMessageObject(params *config.Params, messageID int, msg *Message) *tgbotapi.EditMessageCaptionConfig {
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

func deleteMessageObject(params *config.Params, messageID int) *tgbotapi.DeleteMessageConfig {
	return &tgbotapi.DeleteMessageConfig{
		ChatID:    params.ChatID,
		MessageID: messageID,
	}
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

// Add is add message
func Add(params *config.Params) (tgbotapi.Chattable, error) {
	var Item = params.Item
	imageURL, err := getImageURL(Item.Link)
	if err != nil {
		misc.Error("get_image_url", "get image url", err)
		misc.PushMetrics()
		imageURL = ""
	}
	_, err = url.ParseRequestURI(imageURL)
	if err != nil {
		misc.Error("parse_image_url", "parse image url", err)
		imageURL = ""
	}
	msg, err := createMessageObject(params, &Message{
		FeedTitle:   params.Feed.Title,
		Title:       Item.Title,
		Description: Item.Description,
		Categories:  Item.Categories,
		Link:        Item.Link,
		ImageURL:    imageURL,
	})
	if err != nil {
		misc.Fatal("get_message", "get message", err)
	}
	return msg, nil
}

// Edit is edit message
func Edit(params *config.Params, entry entity.Entry) (*tgbotapi.EditMessageCaptionConfig, error) {
	var Item = params.Item
	imageURL, err := getImageURL(Item.Link)
	if err != nil {
		misc.Error("get_image_url", "get image url", err)
		misc.PushMetrics()
		imageURL = ""
	}
	msg := editMessageObject(params, entry.MessageID, &Message{
		FeedTitle:   params.Feed.Title,
		Title:       Item.Title,
		Description: Item.Description,
		Link:        Item.Link,
		ImageURL:    imageURL,
	})
	return msg, nil
}

// Delete is delete message
func Delete(params *config.Params, entry entity.Entry) error {
	msg := deleteMessageObject(params, entry.MessageID)
	_, err := params.Bot.Send(msg)
	if err != nil {
		return err
	}
	return nil
}

package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func formatText(msg *Message) string {
	return fmt.Sprintf("<b>%s</b>\n\n%s", msg.Title, msg.Description)
}

// CleanUpText return formated test
func CleanUpText(text string) string {
	text = regexp.MustCompile(`<img.*?/>`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`([\x{0020}\x{00a0}\x{1680}\x{180e}\x{2000}-\x{200b}\x{202f}\x{205f}\x{3000}\x{feff}])`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n+\s+`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`window.addEventListener\(.*?(false|true)\);`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(#[\w-]+\s+|\.[\w-]+\s+)+{.*?}`).ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	return text
}

func getText(ctx context.Context, msg *Message) string {
	provider := ctx.Value(config.CtxProviderKey).(*entity.Provider)
	translateLang := ctx.Value(config.CtxTranslateLangKey).(string)
	msg.Title = CleanUpText(msg.Title)
	msg.Description = CleanUpText(msg.Description)
	if provider.Lang == "EST" && translateLang == "ENG" {
		if msg.Title != "" {
			text, err := translate(msg.Title, "et", "en")
			if err != nil {
				misc.Fatal("get_translate", "get translate", err)
			}
			msg.Title = text
		}
		if msg.Description != "" {
			text, err := translate(msg.Description, "et", "en")
			if err != nil {
				misc.Fatal("get_translate", "get translate", err)
			}
			msg.Description = text
		}
	}
	return formatText(msg)
}

func getButton(ctx context.Context, msg *Message) *tgbotapi.InlineKeyboardMarkup {
	provider := ctx.Value(config.CtxProviderKey).(*entity.Provider)
	link := msg.Link
	name := fmt.Sprintf("Read on %s", provider.Name)
	if msg.Paywall {
		name = "💰" + name
	}
	button := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL(name, link)))
	return &button
}

func createMessageObject(ctx context.Context, msg *Message) (tgbotapi.Chattable, error) {
	text := getText(ctx, msg)
	button := getButton(ctx, msg)
	chatID := ctx.Value(config.CtxChatIDKey).(int64)
	var obj tgbotapi.Chattable
	if msg.ImageURL == "" {
		obj = tgbotapi.MessageConfig{
			BaseChat:              tgbotapi.BaseChat{ChatID: chatID, ReplyMarkup: button},
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
		chatID := ctx.Value(config.CtxChatIDKey).(int64)
		obj = tgbotapi.PhotoConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{ChatID: chatID, ReplyMarkup: button},
				File:     file,
			},
			Caption:   text,
			ParseMode: tgbotapi.ModeHTML,
		}

	}
	return obj, nil
}

func editMessageObject(ctx context.Context, messageID int, msg *Message) *tgbotapi.EditMessageCaptionConfig {
	text := getText(ctx, msg)
	button := getButton(ctx, msg)
	chatID := ctx.Value(config.CtxChatIDKey).(int64)
	return &tgbotapi.EditMessageCaptionConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: button,
		},
		Caption:   text,
		ParseMode: tgbotapi.ModeHTML,
	}
}

func deleteMessageObject(ctx context.Context, messageID int) *tgbotapi.DeleteMessageConfig {
	chatID := ctx.Value(config.CtxChatIDKey).(int64)
	return &tgbotapi.DeleteMessageConfig{
		ChatID:    chatID,
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
	Paywall     bool
}

// Add is add message
func Add(ctx context.Context, item *config.FeedItem) (tgbotapi.Chattable, error) {
	feedTitle := ctx.Value(config.CtxFeedTitleKey).(string)
	msg, err := createMessageObject(ctx, &Message{
		FeedTitle:   feedTitle,
		Title:       item.Title,
		Description: item.Description,
		Categories:  item.Categories,
		Link:        item.Link,
		ImageURL:    item.ImageURL,
		Paywall:     item.Paywall,
	})
	if err != nil {
		misc.Fatal("get_message", "get message", err)
	}
	return msg, nil
}

// Edit is edit message
func Edit(ctx context.Context, item *config.FeedItem, entry entity.Entry) (*tgbotapi.EditMessageCaptionConfig, error) {
	feedTitle := ctx.Value(config.CtxFeedTitleKey).(string)
	msg := editMessageObject(ctx, entry.MessageID, &Message{
		FeedTitle:   feedTitle,
		Title:       item.Title,
		Description: item.Description,
		Categories:  item.Categories,
		Link:        item.Link,
		ImageURL:    item.ImageURL,
		Paywall:     item.Paywall,
	})
	return msg, nil
}

// Delete is delete message
func Delete(ctx context.Context, entry entity.Entry) error {
	bot := ctx.Value(config.CtxBotKey).(*tgbotapi.BotAPI)
	msg := deleteMessageObject(ctx, entry.MessageID)
	_, err := bot.Request(msg)
	if err != nil {
		return err
	}
	return nil
}

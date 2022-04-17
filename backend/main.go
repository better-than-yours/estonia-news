package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"estonia-news/config"
	"estonia-news/db"
	"estonia-news/entity"
	"estonia-news/misc"
	"estonia-news/service"
	"estonia-news/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"gorm.io/gorm"
)

func hasChanges(item *config.FeedItem, entry entity.Entry) bool {
	if entry.Title != item.Title || entry.Description != item.Description || entry.Link != item.Link {
		return true
	}
	return false
}

func addRecord(params *config.Params, item *config.FeedItem) error {
	var entry entity.Entry
	result := params.DB.First(&entry, "guid = ?", item.GUID)
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if hasChanges(item, entry) {
			misc.L.Logf("INFO send edit message '%s'", entry.GUID)
			msg, err := service.Edit(params, item, entry)
			if err != nil {
				if strings.Contains(err.Error(), "message to edit not found") {
					if err = service.DeleteRecord(params, entry); err != nil {
						misc.Error("delete_record", fmt.Sprintf("delete record '%s'", entry.GUID), err)
						return err
					}
					time.Sleep(config.TimeoutBetweenMessages)
				}
				return err
			}
			sendedMsg, err := sendMessage(params, msg)
			if err != nil {
				return err
			}
			err = service.UpsertRecord(params, item, sendedMsg.MessageID)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if !isValidItemByTerm(item) {
		return nil
	}
	misc.L.Logf("INFO send message '%s'", item.GUID)
	msg, err := service.Add(params, item)
	if err != nil {
		return err
	}
	sendedMsg, err := sendMessage(params, msg)
	if err != nil {
		return err
	}
	err = service.UpsertRecord(params, item, sendedMsg.MessageID)
	if err != nil {
		return err
	}
	time.Sleep(config.TimeoutBetweenMessages)
	return nil
}

func sendMessage(params *config.Params, msg tgbotapi.Chattable) (*tgbotapi.Message, error) {
	sendedMsg, err := params.Bot.Send(msg)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			misc.Error("send_message", "send message", err)
		} else if strings.Contains(err.Error(), "there is no caption in the message to edit") {
			misc.Error("send_message", "send message", err)
		}
		return nil, err
	}
	return &sendedMsg, nil
}

func getProviders(dbConnect *gorm.DB) []entity.Provider {
	var providers []entity.Provider
	result := dbConnect.Find(&providers)
	if result.Error != nil {
		misc.Fatal("get_providers", "get providers", result.Error)
	}
	return providers
}

func deleteDeletedEntries(params *config.Params, items []*config.FeedItem) error {
	var entries []entity.Entry
	result := params.DB.Where(fmt.Sprintf("published > NOW() - INTERVAL '%d hours' AND provider_id = %d", config.TimeShift/time.Hour, params.Provider.ID)).Find(&entries)
	if result.Error != nil {
		misc.Fatal("query_entries", "query entries", result.Error)
	}
	items = funk.Filter(items, isValidItemByTerm).([]*config.FeedItem)
	for _, entry := range entries {
		foundEntry := funk.Contains(items, func(item *config.FeedItem) bool {
			return entry.GUID == item.GUID
		})
		if !foundEntry {
			if err := service.Delete(params, entry); err != nil {
				if strings.Contains(err.Error(), "message to delete not found") {
					misc.Error("delete_message", fmt.Sprintf("delete message '%s'", entry.GUID), err)
				} else {
					return err
				}
			}
			if err := service.DeleteRecord(params, entry); err != nil {
				misc.Error("delete_record", fmt.Sprintf("delete record '%s'", entry.GUID), err)
				return err
			}
			time.Sleep(config.TimeoutBetweenMessages)
		}
	}
	return nil
}

func isValidItemByContent(params *config.Params, item *config.FeedItem) bool {
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

func isValidItemByTerm(item *config.FeedItem) bool {
	pubDate, _ := time.Parse(time.RFC1123Z, item.Published)
	return !pubDate.Add(config.TimeShift).Before(time.Now())
}

func findSimilarRecord(params *config.Params, item *config.FeedItem) (bool, error) {
	var entry entity.Entry
	result := params.DB.First(&entry, "updated_at > NOW() - INTERVAL '1 day' AND provider_id != ? AND similarity(?,title) > 0.3", params.Provider.ID, item.Title)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func addMissingEntries(params *config.Params, items []*config.FeedItem) error {
	for _, item := range items {
		found, err := findSimilarRecord(params, item)
		if err != nil {
			misc.Error("find_similar_record", fmt.Sprintf("find similar record '%s'", item.GUID), err)
			return err
		}
		if found {
			continue
		}
		if err := addRecord(params, item); err != nil {
			misc.Error("add_record", fmt.Sprintf("add record '%s'", item.GUID), err)
			return err
		}
	}
	return nil
}

func cleanUp(dbConnect *gorm.DB) {
	ticker := time.NewTicker(config.PurgeOldEntriesEvery)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			dbConnect.Unscoped().Where("updated_at < NOW() - INTERVAL '7 days'").Delete(&entity.Entry{})
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func main() {
	_ = godotenv.Load()
	misc.InitMetrics(os.Getenv("PROMETHEUS_URL"), os.Getenv("PROMETHEUS_JOB"))
	dbConnect := db.Connect(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	bot, chatID := telegram.Connect(os.Getenv("TELEGRAM_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
	bot.Debug = os.Getenv("DEBUG") == "true"
	misc.L.Logf("INFO authorized on account '%s'", bot.Self.UserName)

	go cleanUp(dbConnect)
	job(dbConnect, bot, chatID)
	misc.PushMetrics()

	ticker := time.NewTicker(config.TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			job(dbConnect, bot, chatID)
			misc.PushMetrics()
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
		feed, err := service.GetFeed(provider.URL)
		if err != nil {
			misc.Fatal("get_feed", "get feed", err)
		}
		params := &config.Params{
			Bot:               bot,
			DB:                dbConnect,
			Feed:              feed,
			Provider:          provider,
			ChatID:            chatID,
			BlockedCategories: provider.BlockedCategories,
			BlockedWords:      provider.BlockedWords,
		}
		categoriesMap, err := service.AddMissedCategories(params, feed.Items)
		if err != nil {
			misc.Fatal("add_missed_categories", "add missed categories", err)
		}
		items := funk.Map(feed.Items, func(item *gofeed.Item) *config.FeedItem {
			var guid string
			if len(item.GUID) > 0 {
				guid = misc.FormatGUID(item.GUID)
			} else {
				guid = misc.FormatGUID(item.Link)
			}
			categoriesIds := funk.Map(item.Categories, func(category string) int {
				return categoriesMap[category]
			}).([]int)
			return &config.FeedItem{
				GUID:          guid,
				Link:          item.Link,
				Title:         item.Title,
				Description:   item.Description,
				Categories:    item.Categories,
				Published:     item.Published,
				CategoriesIds: categoriesIds,
			}
		}).([]*config.FeedItem)
		items = funk.Filter(items, func(item *config.FeedItem) bool {
			return isValidItemByContent(params, item)
		}).([]*config.FeedItem)
		sort.Slice(items, func(i, j int) bool {
			return items[i].Published > items[j].Published
		})
		if err := deleteDeletedEntries(params, items); err != nil {
			misc.Fatal("delete_record", "delete record", err)
		}
		if err := addMissingEntries(params, items); err != nil {
			misc.Fatal("add_edit_record", "add/edit record", err)
		}
	}
}

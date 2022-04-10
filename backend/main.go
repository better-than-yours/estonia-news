package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thoas/go-funk"
	"gorm.io/gorm"
)

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

func hasChanges(item *gofeed.Item, entry entity.Entry) bool {
	if entry.Title != item.Title || entry.Description != item.Description || entry.Link != item.Link {
		return true
	}
	return false
}

func addRecord(params *config.Params) error {
	var Item = params.Item
	var entry entity.Entry
	result := params.DB.First(&entry, "guid = ?", Item.GUID)
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if hasChanges(Item, entry) {
			misc.L.Logf("INFO send edit message '%s'", formatGUID(entry.GUID))
			msg, err := service.Edit(params, entry)
			if err != nil {
				if strings.Contains(err.Error(), "message to edit not found") {
					if err = deleteRecord(params, entry); err != nil {
						misc.L.Logf("ERROR delete record '%s', %v", formatGUID(entry.GUID), err)
						misc.TaskErrors.With(prometheus.Labels{"error": "delete_record"}).Inc()
						return err
					}
				}
				return err
			}
			err = sendMessage(params, msg)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if !isValidItemByTerm(Item) {
		return nil
	}
	misc.L.Logf("INFO send message '%s'", Item.GUID)
	msg, err := service.Add(params)
	if err != nil {
		return err
	}
	err = sendMessage(params, msg)
	if err != nil {
		return err
	}
	time.Sleep(config.TimeoutBetweenMessages)
	return nil
}

func deleteRecord(params *config.Params, entry entity.Entry) error {
	// TODO need to fix it
	result := params.DB.Unscoped().Where("entry_id = ?", formatGUID(entry.GUID)).Delete(&entity.EntryToCategory{})
	if result.Error != nil {
		return result.Error
	}
	result = params.DB.Unscoped().Where("guid = ?", formatGUID(entry.GUID)).Delete(&entity.Entry{})
	if result.Error != nil {
		return result.Error
	}
	time.Sleep(config.TimeoutBetweenMessages)
	return nil
}

func sendMessage(params *config.Params, msg tgbotapi.Chattable) error {
	sendedMsg, err := params.Bot.Send(msg)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			misc.L.Logf("ERROR send message, %v", err)
			misc.TaskErrors.With(prometheus.Labels{"error": "send_message"}).Inc()
		} else if strings.Contains(err.Error(), "there is no caption in the message to edit") {
			misc.L.Logf("ERROR send message, %v", err)
			misc.TaskErrors.With(prometheus.Labels{"error": "send_message"}).Inc()
		} else {
			return err
		}
	}
	var Item = params.Item
	pubDate, err := time.Parse(time.RFC1123Z, Item.Published)
	if err != nil {
		misc.TaskErrors.With(prometheus.Labels{"error": "parse_date"}).Inc()
		misc.PushMetrics()
		misc.L.Logf("FATAL parse date, %v", err)
	}

	var entry entity.Entry
	result := params.DB.First(&entry, "guid = ?", Item.GUID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		entry = entity.Entry{
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
		var entryToCategory []entity.EntryToCategory
		for _, categoryName := range Item.Categories {
			category := entity.Category{
				Name:     categoryName,
				Provider: params.Provider,
			}
			result = entity.UpsertCategory(params.DB, &category)
			if result.Error != nil {
				return result.Error
			}
			entryToCategory = append(entryToCategory, entity.EntryToCategory{
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

func getProviders(dbConnect *gorm.DB) []entity.Provider {
	var providers []entity.Provider
	result := dbConnect.Find(&providers)
	if result.Error != nil {
		misc.TaskErrors.With(prometheus.Labels{"error": "get_providers"}).Inc()
		misc.PushMetrics()
		misc.L.Logf("FATAL get providers, %v", result.Error)
	}
	return providers
}

func deleteDeletedEntries(params *config.Params, items []*gofeed.Item) error {
	var entries []entity.Entry
	result := params.DB.Where(fmt.Sprintf("published > NOW() - INTERVAL '%d hours' AND provider_id = %d", config.TimeShift, params.Provider.ID)).Find(&entries)
	if result.Error != nil {
		misc.TaskErrors.With(prometheus.Labels{"error": "query_entries"}).Inc()
		misc.PushMetrics()
		misc.L.Logf("FATAL query entries, %v", result.Error)
	}
	items = funk.Filter(items, isValidItemByTerm).([]*gofeed.Item)
	for _, entry := range entries {
		foundEntry := funk.Contains(items, func(item *gofeed.Item) bool {
			return formatGUID(entry.GUID) == item.GUID
		})
		if !foundEntry {
			if err := service.Delete(params, entry); err != nil {
				if strings.Contains(err.Error(), "message to delete not found") {
					misc.L.Logf("ERROR delete message '%s', %v", formatGUID(entry.GUID), err)
					misc.TaskErrors.With(prometheus.Labels{"error": "delete_message"}).Inc()
				} else {
					return err
				}
			}
			if err := deleteRecord(params, entry); err != nil {
				misc.L.Logf("ERROR delete record '%s', %v", formatGUID(entry.GUID), err)
				misc.TaskErrors.With(prometheus.Labels{"error": "delete_record"}).Inc()
				return err
			}
		}
	}
	return nil
}

func isValidItemByContent(params *config.Params, item *gofeed.Item) bool {
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
	return !pubDate.Add(time.Duration(config.TimeShift) * time.Hour).Before(time.Now())
}

func findSimilarRecord(params *config.Params, item *gofeed.Item) (bool, error) {
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

func addMissingEntries(params *config.Params, items []*gofeed.Item) error {
	for _, item := range items {
		found, err := findSimilarRecord(params, item)
		if err != nil {
			misc.L.Logf("ERROR find similar record '%s', %v", item.GUID, err)
			misc.TaskErrors.With(prometheus.Labels{"error": "find_similar_record"}).Inc()
			return err
		}
		if found {
			continue
		}
		params.Item = item
		if err := addRecord(params); err != nil {
			misc.L.Logf("ERROR add record '%s', %v", item.GUID, err)
			misc.TaskErrors.With(prometheus.Labels{"error": "add_record"}).Inc()
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
			dbConnect.Unscoped().Select("Entry").Where("entries.updated_at < NOW() - INTERVAL '7 days'").Delete(&entity.EntryToCategory{})
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
			misc.TaskErrors.With(prometheus.Labels{"error": "get_feed"}).Inc()
			misc.PushMetrics()
			misc.L.Logf("FATAL get feed, %v", err)
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
			misc.TaskErrors.With(prometheus.Labels{"error": "delete_record"}).Inc()
			misc.PushMetrics()
			misc.L.Logf("FATAL delete record, %v", err)
		}
		if err := addMissingEntries(params, feed.Items); err != nil {
			misc.TaskErrors.With(prometheus.Labels{"error": "add_edit_record"}).Inc()
			misc.PushMetrics()
			misc.L.Logf("FATAL add/edit record, %v", err)
		}
	}
}

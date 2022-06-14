package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"estonia-news/command"
	"estonia-news/config"
	"estonia-news/db"
	"estonia-news/entity"
	"estonia-news/misc"
	"estonia-news/service"
	"estonia-news/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"github.com/uptrace/bun"
)

func hasChanges(item *config.FeedItem, entry entity.Entry) bool {
	if entry.Title == item.Title && entry.Description == item.Description && entry.Link == item.Link {
		return false
	}
	return true
}

func checkRecord(ctx context.Context, item *config.FeedItem) error {
	var entry entity.Entry
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	err := dbConnect.NewSelect().Model(&entry).Where("id = ?", item.GUID).Limit(1).Scan(ctx)
	if !errors.Is(err, sql.ErrNoRows) {
		if hasChanges(item, entry) {
			if err := editMessage(ctx, item, entry); err != nil {
				return err
			}
		}
		return nil
	}
	if !isValidItemByTerm(item) {
		return nil
	}
	if err := newMessage(ctx, item); err != nil {
		return err
	}
	time.Sleep(config.TimeoutBetweenMessages)
	return nil
}

func editMessage(ctx context.Context, item *config.FeedItem, entry entity.Entry) error {
	misc.L.Logf("INFO send edit message '%s'", entry.ID)
	msg, err := service.Edit(ctx, item, entry)
	if err != nil {
		if strings.Contains(err.Error(), "message to edit not found") {
			if err = service.DeleteRecord(ctx, entry); err != nil {
				misc.Error("delete_record", fmt.Sprintf("delete record '%s'", entry.ID), err)
				return err
			}
			time.Sleep(config.TimeoutBetweenMessages)
		}
		return err
	}
	_, err = sendMessage(ctx, msg)
	if err != nil {
		return err
	}
	err = service.UpsertRecord(ctx, item, entry.MessageID)
	if err != nil {
		return err
	}
	return nil
}

func newMessage(ctx context.Context, item *config.FeedItem) error {
	misc.L.Logf("INFO send message '%s'", item.GUID)
	msg, err := service.Add(ctx, item)
	if err != nil {
		return err
	}
	sendedMsg, err := sendMessage(ctx, msg)
	if err != nil {
		return err
	}
	if sendedMsg.MessageID == 0 {
		err = errors.New("empty MessageID")
		misc.Error("add_record", fmt.Sprintf("add record '%s'", item.GUID), err)
		return err
	}
	err = service.UpsertRecord(ctx, item, sendedMsg.MessageID)
	if err != nil {
		return err
	}
	return nil
}

func sendMessage(ctx context.Context, msg tgbotapi.Chattable) (*tgbotapi.Message, error) {
	bot := ctx.Value(config.CtxBotKey).(*tgbotapi.BotAPI)
	sendedMsg, err := bot.Send(msg)
	if err != nil {
		if funk.Contains([]string{"message is not modified", "there is no caption in the message to edit"}, func(item string) bool {
			return strings.Contains(err.Error(), item)
		}) {
			misc.Error("send_message", "send message", err)
			return &sendedMsg, nil
		}
		return nil, err
	}
	return &sendedMsg, nil
}

func getProviders(ctx context.Context) []entity.Provider {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	var providers []entity.Provider
	err := dbConnect.NewSelect().Model(&providers).Scan(ctx)
	if err != nil {
		misc.Fatal("get_providers", "get providers", err)
	}
	return providers
}

func deleteDeletedEntries(ctx context.Context, items []*config.FeedItem) error {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	provider := ctx.Value(config.CtxProviderKey).(*entity.Provider)
	var entries []entity.Entry
	err := dbConnect.NewSelect().Model(&entries).Where(fmt.Sprintf("published_at > NOW() - INTERVAL '%d hours' AND provider_id = %d", config.TimeShift/time.Hour, provider.ID)).Scan(ctx)
	if err != nil {
		misc.Fatal("query_entries", "query entries", err)
	}
	items = funk.Filter(items, isValidItemByTerm).([]*config.FeedItem)
	for _, entry := range entries {
		foundEntry := funk.Contains(items, func(item *config.FeedItem) bool {
			return entry.ID == item.GUID
		})
		if !foundEntry {
			if err := service.Delete(ctx, entry); err != nil {
				misc.Error("delete_message", fmt.Sprintf("delete message '%s'", entry.ID), err)
				if !strings.Contains(err.Error(), "message to delete not found") {
					return err
				}
			}
			if err := service.DeleteRecord(ctx, entry); err != nil {
				misc.Error("delete_record", fmt.Sprintf("delete record '%s'", entry.ID), err)
				return err
			}
			time.Sleep(config.TimeoutBetweenMessages)
		}
	}
	return nil
}

func isValidItemByContent(ctx context.Context, blockedCategories, blockedWords []string, item *config.FeedItem) bool {
	if len(funk.IntersectString(blockedCategories, item.Categories)) > 0 {
		return false
	}
	foundBlockedWords := funk.FilterString(blockedWords, func(word string) bool {
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

func findSimilarRecord(ctx context.Context, item *config.FeedItem) (bool, error) {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	provider := ctx.Value(config.CtxProviderKey).(*entity.Provider)
	var entry entity.Entry
	exists, err := dbConnect.NewSelect().Model(&entry).Where("updated_at > NOW() - INTERVAL '1 day' AND provider_id != ? AND similarity(?,title) > 0.3", provider.ID, item.Title).Exists(ctx)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func addMissingEntries(ctx context.Context, items []*config.FeedItem) error {
	for _, item := range items {
		found, err := findSimilarRecord(ctx, item)
		if err != nil {
			misc.Error("find_similar_record", fmt.Sprintf("find similar record '%s'", item.GUID), err)
			return err
		}
		if found {
			continue
		}
		if err := checkRecord(ctx, item); err != nil {
			misc.Error("check_record", fmt.Sprintf("check record '%s'", item.GUID), err)
			return err
		}
	}
	return nil
}

func cleanUp(ctx context.Context) {
	ticker := time.NewTicker(config.PurgeOldEntriesEvery)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
			_, _ = dbConnect.NewDelete().Model(&entity.Entry{}).Where("updated_at < NOW() - INTERVAL '7 days'").Exec(ctx)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func pushMetrics() {
	ticker := time.NewTicker(config.PushMetricsEvery)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			misc.PushMetrics()
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func main() {
	_ = godotenv.Load()
	ctx := context.Background()
	misc.InitMetrics(os.Getenv("PROMETHEUS_URL"), os.Getenv("PROMETHEUS_JOB"))
	dbConnect := db.Connect(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	ctx = context.WithValue(ctx, config.CtxDBKey, dbConnect)
	bot := telegram.Connect(os.Getenv("TELEGRAM_TOKEN"))
	ctx = context.WithValue(ctx, config.CtxBotKey, bot)
	bot.Debug = os.Getenv("DEBUG") == "true"
	misc.L.Logf("INFO authorized on account '%s'", bot.Self.UserName)
	commander := os.Getenv("COMMANDER")
	go pushMetrics()
	if len(commander) > 0 {
		db.Migrate(ctx)
		handleCommand(ctx, commander)
	} else {
		handleNews(ctx)
	}
}

func handleCommand(ctx context.Context, commander string) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	bot := ctx.Value(config.CtxBotKey).(*tgbotapi.BotAPI)
	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		isValidCommand := update.Message.IsCommand() && update.Message != nil && update.Message.From.UserName == commander
		if isValidCommand {
			command.ExecCommand(ctx, update.Message)
		}
	}
}

func handleNews(ctx context.Context) {
	chatID, err := strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)
	if err != nil {
		misc.Fatal("tg_chat_id", "chat id", err)
	}
	ctx = context.WithValue(ctx, config.CtxChatIDKey, chatID)
	go cleanUp(ctx)
	job(ctx, chatID)
	ticker := time.NewTicker(config.TimeoutBetweenLoops)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			job(ctx, chatID)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func job(ctx context.Context, chatID int64) {
	providers := getProviders(ctx)
	for _, provider := range providers {
		if provider.Lang != os.Getenv("LANG_NEWS") {
			continue
		}
		feed, err := service.GetFeed(provider.URL)
		if err != nil {
			misc.Fatal("get_feed", "get feed", err)
		}
		ctx = context.WithValue(ctx, config.CtxProviderKey, &provider) //nolint:gosec,gocritic
		ctx = context.WithValue(ctx, config.CtxFeedTitleKey, feed.Title)
		blocks, err := entity.GetListBlocks(ctx)
		if err != nil {
			misc.Fatal("get_blocked_categories", "get blocked categories", err)
		}
		blocks = funk.Filter(blocks, func(item entity.BlockedCategory) bool {
			return item.Category.ProviderID == provider.ID
		}).([]entity.BlockedCategory)
		blockedCategories := funk.Map(blocks, func(item entity.BlockedCategory) string {
			return item.Category.Name
		}).([]string)
		categoriesMap, err := service.AddMissedCategories(ctx, feed.Items)
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
			return isValidItemByContent(ctx, blockedCategories, provider.BlockedWords, item)
		}).([]*config.FeedItem)
		sort.Slice(items, func(i, j int) bool {
			return items[i].Published > items[j].Published
		})
		if err := deleteDeletedEntries(ctx, items); err != nil {
			misc.Fatal("delete_record", "delete record", err)
		}
		if err := addMissingEntries(ctx, items); err != nil {
			misc.Fatal("add_edit_record", "add/edit record", err)
		}
	}
}

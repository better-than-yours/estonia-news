package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"

	"github.com/uptrace/bun"
)

// DeleteRecord perform delete record
func DeleteRecord(ctx context.Context, entry entity.Entry) error {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	_, err := dbConnect.NewDelete().Model(&entry).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete record '%s': %v", entry.ID, err)
	}
	return nil
}

// UpsertRecord perform add/update record
func UpsertRecord(ctx context.Context, item *config.FeedItem, messageID int) error {
	pubDate, err := time.Parse(time.RFC1123Z, item.Published)
	if err != nil {
		misc.Fatal("parse_date", "parse date", err)
	}
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	provider := ctx.Value(config.CtxProviderKey).(*entity.Provider)
	entry := entity.Entry{
		ID:          item.GUID,
		ProviderID:  provider.ID,
		Link:        item.Link,
		Title:       item.Title,
		Description: item.Description,
		ImageURL:    item.ImageURL,
		Paywall:     item.Paywall,
		PublishedAt: pubDate,
		UpdatedAt:   time.Now(),
		MessageID:   messageID,
	}
	_, err = dbConnect.NewInsert().Model(&entry).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add/update record '%s': %v", entry.ID, err)
	}
	_, err = dbConnect.NewDelete().Model(&entity.EntryToCategory{}).Where("entry_id = ?", item.GUID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add/update record '%s': %v", entry.ID, err)
	}
	for _, categoryName := range item.Categories {
		var category entity.Category
		err = dbConnect.NewSelect().Model(&category).Where("name = ? AND provider_id = ?", categoryName, provider.ID).Limit(1).Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				category = entity.Category{
					Name:       categoryName,
					ProviderID: provider.ID,
				}
				_, err = dbConnect.NewInsert().Model(&category).On("CONFLICT (name, provider_id) DO NOTHING").Exec(ctx)
				if err != nil {
					return fmt.Errorf("failed to add/update categories for record '%s': %v", entry.ID, err)
				}
			} else {
				return fmt.Errorf("failed to add/update categories for record '%s': %v", entry.ID, err)
			}
		}
		_, err = dbConnect.NewInsert().Model(&entity.EntryToCategory{
			EntryID:    entry.ID,
			CategoryID: category.ID,
		}).On("CONFLICT (entry_id, category_id) DO NOTHING").Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to add/update categories for record '%s': %v", entry.ID, err)
		}
	}
	return nil
}

package service

import (
	"time"

	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"
)

// DeleteRecord perform delete record
func DeleteRecord(params *config.Params, entry entity.Entry) error {
	result := params.DB.Unscoped().Where("guid = ?", misc.FormatGUID(entry.GUID)).Delete(&entity.Entry{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// UpsertRecord perform add/update record
func UpsertRecord(params *config.Params, messageID int) error {
	var item = params.Item
	pubDate, err := time.Parse(time.RFC1123Z, item.Published)
	if err != nil {
		misc.Fatal("parse_date", "parse date", err)
	}
	entry := entity.Entry{
		GUID:        item.GUID,
		Provider:    params.Provider,
		Link:        item.Link,
		Title:       item.Title,
		Description: item.Description,
		Published:   pubDate,
		MessageID:   messageID,
	}
	result := entity.UpsertEntry(params.DB, &entry)
	if result.Error != nil {
		return result.Error
	}
	for _, categoryName := range item.Categories {
		category := entity.Category{
			Name:     categoryName,
			Provider: params.Provider,
		}
		result = entity.UpsertCategory(params.DB, &category)
		if result.Error != nil {
			return result.Error
		}
		result = entity.UpsertEntryToCategory(params.DB, &entity.EntryToCategory{
			Entry:    entry,
			Category: category,
		})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

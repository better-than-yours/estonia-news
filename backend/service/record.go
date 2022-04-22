package service

import (
	"time"

	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"
)

// DeleteRecord perform delete record
func DeleteRecord(params *config.Params, entry entity.Entry) error {
	result := params.DB.Unscoped().Where("guid = ?", entry.GUID).Delete(&entity.Entry{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// UpsertRecord perform add/update record
func UpsertRecord(params *config.Params, item *config.FeedItem, messageID int) error {
	pubDate, err := time.Parse(time.RFC1123Z, item.Published)
	if err != nil {
		misc.Fatal("parse_date", "parse date", err)
	}
	entry := entity.Entry{
		GUID:        item.GUID,
		ProviderID:  params.Provider.ID,
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
	result = params.DB.Unscoped().Where("entry_id = ?", item.GUID).Delete(&entity.EntryToCategory{})
	if result.Error != nil {
		return result.Error
	}
	for _, categoryName := range item.Categories {
		category := entity.Category{
			Name:       categoryName,
			ProviderID: params.Provider.ID,
		}
		result = params.DB.Where(category).FirstOrCreate(&category)
		if result.Error != nil {
			return result.Error
		}
		entryToCategory := entity.EntryToCategory{
			EntryID:    entry.GUID,
			CategoryID: category.ID,
		}
		result = params.DB.Where(entryToCategory).FirstOrCreate(&entryToCategory)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

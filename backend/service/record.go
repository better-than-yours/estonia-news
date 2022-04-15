package service

import (
	"errors"
	"time"

	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"

	"gorm.io/gorm"
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
	var entry entity.Entry
	result := params.DB.First(&entry, "guid = ?", item.GUID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		entry = entity.Entry{
			GUID:        item.GUID,
			Provider:    params.Provider,
			Link:        item.Link,
			Title:       item.Title,
			Description: item.Description,
			Published:   pubDate,
			MessageID:   messageID,
		}
		result = params.DB.Create(&entry)
		if result.Error != nil {
			return result.Error
		}
		var entryToCategory []entity.EntryToCategory
		for _, categoryName := range item.Categories {
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
		entry.Title = item.Title
		entry.Description = item.Description
		entry.Link = item.Link
		result = params.DB.Where("guid = ?", item.GUID).Updates(&entry)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

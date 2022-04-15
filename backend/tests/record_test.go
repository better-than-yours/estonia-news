package tests

import (
	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/service"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
)

func (t *SuiteTest) Test_Record_DeleteRecord() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)
	entry := []entity.Entry{{GUID: "err#123", Provider: provider}, {GUID: "err#321", Provider: provider}}
	t.db.Create(&entry)
	category := []entity.Category{{Name: "cat1", Provider: provider}, {Name: "cat2", Provider: provider}}
	t.db.Create(&category)
	entryToCategory := []entity.EntryToCategory{{Entry: entry[0], Category: category[0]}, {Entry: entry[1], Category: category[1]}}
	t.db.Create(&entryToCategory)

	err := service.DeleteRecord(&config.Params{DB: t.db}, entity.Entry{GUID: "err#123"})
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		result1 := t.db.Find(&entries)
		assert.EqualValues(t.T(), 1, result1.RowsAffected)
		assert.NoError(t.T(), result1.Error)
		assert.Equal(t.T(), "err#321", entries[0].GUID)

		var entriesToCategories []entity.EntryToCategory
		result2 := t.db.Find(&entriesToCategories)
		assert.EqualValues(t.T(), 1, result2.RowsAffected)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), "err#321", entriesToCategories[0].EntryID)
	}
}

func (t *SuiteTest) Test_Record_UpsertRecord_Create() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)

	err := service.UpsertRecord(&config.Params{
		DB:       t.db,
		Provider: provider,
		Item: &gofeed.Item{
			GUID:        "err#123",
			Published:   "Mon, 02 Jan 2006 15:04:05 -0700",
			Link:        "link",
			Title:       "title",
			Description: "description",
			Categories:  []string{"cat1", "cat3"},
		},
	}, 123)
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		result1 := t.db.Find(&entries)
		assert.EqualValues(t.T(), 1, result1.RowsAffected)
		assert.NoError(t.T(), result1.Error)
		assert.Equal(t.T(), "err#123", entries[0].GUID)

		var categories []entity.Category
		result2 := t.db.Find(&categories)
		assert.EqualValues(t.T(), 2, result2.RowsAffected)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), "cat1", categories[0].Name)
		assert.Equal(t.T(), "cat3", categories[1].Name)

		var entriesToCategories []entity.EntryToCategory
		result3 := t.db.Find(&entriesToCategories)
		assert.EqualValues(t.T(), 2, result3.RowsAffected)
		assert.NoError(t.T(), result3.Error)
		assert.Equal(t.T(), "err#123", entriesToCategories[0].EntryID)
		assert.Equal(t.T(), "err#123", entriesToCategories[1].EntryID)
	}
}

func (t *SuiteTest) Test_Record_UpsertRecord_Update() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)

	err := service.UpsertRecord(&config.Params{
		DB:       t.db,
		Provider: provider,
		Item: &gofeed.Item{
			GUID:        "err#123",
			Published:   "Mon, 02 Jan 2006 15:04:05 -0700",
			Link:        "link",
			Title:       "title",
			Description: "description",
			Categories:  []string{"cat1", "cat3"},
		},
	}, 123)
	assert.NoError(t.T(), err)
	err = service.UpsertRecord(&config.Params{
		DB:       t.db,
		Provider: provider,
		Item: &gofeed.Item{
			GUID:  "err#123",
			Link:  "link_new",
			Title: "title",
		},
	}, 123)
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		result1 := t.db.Find(&entries)
		assert.EqualValues(t.T(), 1, result1.RowsAffected)
		assert.NoError(t.T(), result1.Error)
		assert.Equal(t.T(), "err#123", entries[0].GUID)
		assert.Equal(t.T(), "link_new", entries[0].Link)
		assert.Equal(t.T(), "title", entries[0].Title)

		var categories []entity.Category
		result2 := t.db.Find(&categories)
		assert.EqualValues(t.T(), 2, result2.RowsAffected)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), "cat1", categories[0].Name)
		assert.Equal(t.T(), "cat3", categories[1].Name)

		var entriesToCategories []entity.EntryToCategory
		result3 := t.db.Find(&entriesToCategories)
		assert.EqualValues(t.T(), 2, result3.RowsAffected)
		assert.NoError(t.T(), result3.Error)
		assert.Equal(t.T(), "err#123", entriesToCategories[0].EntryID)
		assert.Equal(t.T(), "err#123", entriesToCategories[1].EntryID)
	}
}

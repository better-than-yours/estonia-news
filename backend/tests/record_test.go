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
		assert.EqualValues(t.T(), result1.RowsAffected, 1)
		assert.NoError(t.T(), result1.Error)
		assert.Equal(t.T(), entries[0].GUID, "err#321")

		var entriesToCategories []entity.EntryToCategory
		result2 := t.db.Find(&entriesToCategories)
		assert.EqualValues(t.T(), result2.RowsAffected, 1)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), entriesToCategories[0].EntryID, "err#321")
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
		assert.EqualValues(t.T(), result1.RowsAffected, 1)
		assert.NoError(t.T(), result1.Error)
		assert.Equal(t.T(), entries[0].GUID, "err#123")

		var categories []entity.Category
		result2 := t.db.Find(&categories)
		assert.EqualValues(t.T(), result2.RowsAffected, 2)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), categories[0].Name, "cat1")
		assert.Equal(t.T(), categories[1].Name, "cat3")

		var entriesToCategories []entity.EntryToCategory
		result3 := t.db.Find(&entriesToCategories)
		assert.EqualValues(t.T(), result3.RowsAffected, 2)
		assert.NoError(t.T(), result3.Error)
		assert.Equal(t.T(), entriesToCategories[0].EntryID, "err#123")
		assert.Equal(t.T(), entriesToCategories[1].EntryID, "err#123")
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
			GUID: "err#123",
			Link: "link_new",
		},
	}, 123)
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		result1 := t.db.Find(&entries)
		assert.EqualValues(t.T(), result1.RowsAffected, 1)
		assert.NoError(t.T(), result1.Error)
		assert.Equal(t.T(), entries[0].GUID, "err#123")
		assert.Equal(t.T(), entries[0].Link, "link_new")
		assert.Equal(t.T(), entries[0].Title, "title")

		var categories []entity.Category
		result2 := t.db.Find(&categories)
		assert.EqualValues(t.T(), result2.RowsAffected, 2)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), categories[0].Name, "cat1")
		assert.Equal(t.T(), categories[1].Name, "cat3")

		var entriesToCategories []entity.EntryToCategory
		result3 := t.db.Find(&entriesToCategories)
		assert.EqualValues(t.T(), result3.RowsAffected, 2)
		assert.NoError(t.T(), result3.Error)
		assert.Equal(t.T(), entriesToCategories[0].EntryID, "err#123")
		assert.Equal(t.T(), entriesToCategories[1].EntryID, "err#123")
	}
}

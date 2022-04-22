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
	entryToCategory := []entity.EntryToCategory{{EntryID: entry[0].GUID, CategoryID: category[0].ID}, {EntryID: entry[1].GUID, CategoryID: category[1].ID}}
	t.db.Create(&entryToCategory)

	err := service.DeleteRecord(&config.Params{DB: t.db}, entity.Entry{GUID: "err#123"})
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		result1 := t.db.Find(&entries)
		assert.EqualValues(t.T(), 1, result1.RowsAffected)
		assert.NoError(t.T(), result1.Error)
		assert.Equal(t.T(), "err#321", entries[0].GUID)

		var entryToCategories []entity.EntryToCategory
		result2 := t.db.Find(&entryToCategories)
		assert.EqualValues(t.T(), 1, result2.RowsAffected)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), "err#321", entryToCategories[0].EntryID)
	}
}

func (t *SuiteTest) Test_Record_UpsertRecord_Create() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)

	err := service.UpsertRecord(&config.Params{
		DB:       t.db,
		Provider: provider,
	}, &config.FeedItem{
		GUID:        "err#123",
		Published:   "Mon, 02 Jan 2006 15:04:05 -0700",
		Link:        "link",
		Title:       "title",
		Description: "description",
		Categories:  []string{"cat1", "cat3"},
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

		var entryToCategories []entity.EntryToCategory
		result3 := t.db.Find(&entryToCategories)
		assert.EqualValues(t.T(), 2, result3.RowsAffected)
		assert.NoError(t.T(), result3.Error)
		assert.Equal(t.T(), "err#123", entryToCategories[0].EntryID)
		assert.Equal(t.T(), "err#123", entryToCategories[1].EntryID)
	}
}

func (t *SuiteTest) Test_Record_UpsertRecord_Update() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)

	err := service.UpsertRecord(&config.Params{
		DB:       t.db,
		Provider: provider,
	}, &config.FeedItem{
		GUID:        "err#123",
		Published:   "Mon, 02 Jan 2006 15:04:05 -0700",
		Link:        "link",
		Title:       "title",
		Description: "description",
		Categories:  []string{"cat1", "cat3"},
	}, 123)
	assert.NoError(t.T(), err)
	err = service.UpsertRecord(&config.Params{
		DB:       t.db,
		Provider: provider,
	}, &config.FeedItem{
		GUID:       "err#123",
		Link:       "link_new",
		Title:      "title",
		Categories: []string{"cat1", "cat2"},
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
		assert.EqualValues(t.T(), 3, result2.RowsAffected)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), "cat1", categories[0].Name)
		assert.Equal(t.T(), "cat3", categories[1].Name)
		assert.Equal(t.T(), "cat2", categories[2].Name)

		var entryToCategories []entity.EntryToCategory
		result3 := t.db.Find(&entryToCategories)
		assert.EqualValues(t.T(), 2, result3.RowsAffected)
		assert.NoError(t.T(), result3.Error)
		assert.Equal(t.T(), "err#123", entryToCategories[0].EntryID)
		assert.Equal(t.T(), "err#123", entryToCategories[1].EntryID)
		assert.Equal(t.T(), 1, entryToCategories[0].CategoryID)
		assert.Equal(t.T(), 3, entryToCategories[1].CategoryID)
	}
}

func (t *SuiteTest) Test_Record_AddMissedCategories() {
	providers := []entity.Provider{{URL: "err.ee"}, {URL: "pm.ee"}}
	t.db.Create(&providers)

	categoriesMap, err := service.AddMissedCategories(&config.Params{
		DB:       t.db,
		Provider: providers[0],
	}, []*gofeed.Item{{
		Categories: []string{"cat1", "cat2"},
	}, {
		Categories: []string{"cat1", "cat3"},
	}})
	if assert.NoError(t.T(), err) {
		var categories []entity.Category
		result2 := t.db.Find(&categories)
		assert.EqualValues(t.T(), 3, result2.RowsAffected)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), "cat1", categories[0].Name)
		assert.Equal(t.T(), "cat2", categories[1].Name)
		assert.Equal(t.T(), "cat3", categories[2].Name)
		assert.Equal(t.T(), providers[0].ID, categories[0].ProviderID)
		assert.Equal(t.T(), providers[0].ID, categories[1].ProviderID)
		assert.Equal(t.T(), providers[0].ID, categories[2].ProviderID)
		assert.Equal(t.T(), 1, categoriesMap["cat1"])
		assert.Equal(t.T(), 2, categoriesMap["cat2"])
		assert.Equal(t.T(), 3, categoriesMap["cat3"])
	}

	categoriesMap, err = service.AddMissedCategories(&config.Params{
		DB:       t.db,
		Provider: providers[1],
	}, []*gofeed.Item{{
		Categories: []string{"cat1", "cat3"},
	}, {
		Categories: []string{"cat1"},
	}})
	if assert.NoError(t.T(), err) {
		var categories []entity.Category
		result2 := t.db.Find(&categories)
		assert.EqualValues(t.T(), 5, result2.RowsAffected)
		assert.NoError(t.T(), result2.Error)
		assert.Equal(t.T(), "cat1", categories[3].Name)
		assert.Equal(t.T(), "cat3", categories[4].Name)
		assert.Equal(t.T(), providers[1].ID, categories[3].ProviderID)
		assert.Equal(t.T(), providers[1].ID, categories[4].ProviderID)
		assert.Equal(t.T(), 4, categoriesMap["cat1"])
		assert.Equal(t.T(), 5, categoriesMap["cat3"])
	}
}

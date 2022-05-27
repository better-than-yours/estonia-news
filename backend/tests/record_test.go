package tests

import (
	"context"

	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/service"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
)

func (t *SuiteTest) Test_Record_DeleteRecord() {
	LoadFixtures(t)
	err := service.DeleteRecord(t.ctx, entity.Entry{ID: "err#123"})
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		err := t.db.NewSelect().Model(&entries).Scan(t.ctx)
		assert.NoError(t.T(), err)
		assert.EqualValues(t.T(), 1, len(entries))
		assert.Equal(t.T(), "err#321", entries[0].ID)

		var entryToCategories []entity.EntryToCategory
		err = t.db.NewSelect().Model(&entryToCategories).Scan(t.ctx)
		assert.NoError(t.T(), err)
		assert.EqualValues(t.T(), 1, len(entryToCategories))
		assert.Equal(t.T(), "err#321", entryToCategories[0].EntryID)
	}
}

func (t *SuiteTest) Test_Record_UpsertRecord_Create() {
	LoadFixtures(t)
	err := service.UpsertRecord(t.ctx, &config.FeedItem{
		GUID:        "pm#123",
		Published:   "Mon, 02 Jan 2006 15:04:05 -0700",
		Link:        "link",
		Title:       "title",
		Description: "description",
		Categories:  []string{"cat1", "cat3"},
	}, 123)
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		err = t.db.NewSelect().Model(&entries).Scan(t.ctx)
		assert.EqualValues(t.T(), 3, len(entries))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "pm#123", entries[2].ID)

		var categories []entity.Category
		err = t.db.NewSelect().Model(&categories).Scan(t.ctx)
		assert.EqualValues(t.T(), 3, len(categories))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "cat1", categories[0].Name)
		assert.Equal(t.T(), "cat2", categories[1].Name)
		assert.Equal(t.T(), "cat3", categories[2].Name)

		var entryToCategories []entity.EntryToCategory
		err = t.db.NewSelect().Model(&entryToCategories).Scan(t.ctx)
		assert.EqualValues(t.T(), 5, len(entryToCategories))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "err#123", entryToCategories[0].EntryID)
		assert.Equal(t.T(), "err#123", entryToCategories[1].EntryID)
		assert.Equal(t.T(), "err#321", entryToCategories[2].EntryID)
		assert.Equal(t.T(), "pm#123", entryToCategories[3].EntryID)
		assert.Equal(t.T(), "pm#123", entryToCategories[4].EntryID)
	}
}

func (t *SuiteTest) Test_Record_UpsertRecord_Update() {
	LoadFixtures(t)
	err := service.UpsertRecord(t.ctx, &config.FeedItem{
		GUID:        "pm#123",
		Published:   "Mon, 02 Jan 2006 15:04:05 -0700",
		Link:        "link",
		Title:       "title",
		Description: "description",
		Categories:  []string{"cat1", "cat3"},
	}, 123)
	assert.NoError(t.T(), err)
	err = service.UpsertRecord(t.ctx, &config.FeedItem{
		GUID:       "pm#123",
		Link:       "link_new",
		Published:  "Mon, 02 Jan 2006 15:04:05 -0700",
		Title:      "title",
		Categories: []string{"cat1", "cat2"},
	}, 123)
	if assert.NoError(t.T(), err) {
		var entries []entity.Entry
		err = t.db.NewSelect().Model(&entries).Scan(t.ctx)
		assert.EqualValues(t.T(), 3, len(entries))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "pm#123", entries[2].ID)
		assert.Equal(t.T(), "link_new", entries[2].Link)
		assert.Equal(t.T(), "title", entries[2].Title)

		var categories []entity.Category
		err = t.db.NewSelect().Model(&categories).Scan(t.ctx)
		assert.EqualValues(t.T(), 3, len(categories))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "cat1", categories[0].Name)
		assert.Equal(t.T(), "cat2", categories[1].Name)
		assert.Equal(t.T(), "cat3", categories[2].Name)

		var entryToCategories []entity.EntryToCategory
		err = t.db.NewSelect().Model(&entryToCategories).Relation("Category").Scan(t.ctx)
		assert.EqualValues(t.T(), 5, len(entryToCategories))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "pm#123", entryToCategories[3].EntryID)
		assert.Equal(t.T(), "pm#123", entryToCategories[4].EntryID)
		assert.Equal(t.T(), "cat1", entryToCategories[3].Category.Name)
		assert.Equal(t.T(), "cat2", entryToCategories[4].Category.Name)
	}
}

func (t *SuiteTest) Test_Record_AddMissedCategories() {
	LoadFixtures(t)
	var providers []entity.Provider
	_ = t.db.NewSelect().Model(&providers).Scan(t.ctx)
	categoriesMap, err := service.AddMissedCategories(t.ctx, []*gofeed.Item{{
		Categories: []string{"cat1", "cat2"},
	}, {
		Categories: []string{"cat1", "cat3"},
	}})
	if assert.NoError(t.T(), err) {
		var categories []entity.Category
		err = t.db.NewSelect().Model(&categories).Relation("Provider").Scan(t.ctx)
		assert.EqualValues(t.T(), 3, len(categories))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "cat1", categories[0].Name)
		assert.Equal(t.T(), "cat2", categories[1].Name)
		assert.Equal(t.T(), "cat3", categories[2].Name)
		assert.Equal(t.T(), providers[0].ID, categories[0].Provider.ID)
		assert.Equal(t.T(), providers[0].ID, categories[1].Provider.ID)
		assert.Equal(t.T(), providers[0].ID, categories[2].Provider.ID)
		assert.Equal(t.T(), categories[0].ID, categoriesMap["cat1"])
		assert.Equal(t.T(), categories[1].ID, categoriesMap["cat2"])
		assert.Equal(t.T(), categories[2].ID, categoriesMap["cat3"])
	}

	t.ctx = context.WithValue(t.ctx, config.CtxProviderKey, &providers[1])
	categoriesMap, err = service.AddMissedCategories(t.ctx, []*gofeed.Item{{
		Categories: []string{"cat1", "cat3"},
	}, {
		Categories: []string{"cat1"},
	}})
	if assert.NoError(t.T(), err) {
		var categories []entity.Category
		err = t.db.NewSelect().Model(&categories).Relation("Provider").Scan(t.ctx)
		assert.EqualValues(t.T(), 5, len(categories))
		assert.NoError(t.T(), err)
		assert.Equal(t.T(), "cat1", categories[3].Name)
		assert.Equal(t.T(), "cat3", categories[4].Name)
		assert.Equal(t.T(), providers[1].ID, categories[3].Provider.ID)
		assert.Equal(t.T(), providers[1].ID, categories[4].Provider.ID)
		assert.Equal(t.T(), categories[3].ID, categoriesMap["cat1"])
		assert.Equal(t.T(), categories[4].ID, categoriesMap["cat3"])
	}
}

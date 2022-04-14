package tests

import (
	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/service"

	"github.com/stretchr/testify/assert"
)

func (t *SuiteTest) TestDeleteRecord() {
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

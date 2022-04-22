package tests

import (
	"estonia-news/entity"

	"github.com/stretchr/testify/assert"
)

func (t *SuiteTest) Test_Command_GetEntryByID() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)
	entry := []entity.Entry{{GUID: "err#123", Provider: provider}, {GUID: "err#321", Provider: provider}}
	t.db.Create(&entry)
	category := []entity.Category{{Name: "cat1", Provider: provider}, {Name: "cat2", Provider: provider}}
	t.db.Create(&category)
	entryToCategory := []entity.EntryToCategory{{EntryID: entry[0].GUID, CategoryID: category[0].ID}, {EntryID: entry[0].GUID, CategoryID: category[1].ID}, {EntryID: entry[1].GUID, CategoryID: category[1].ID}}
	t.db.Create(&entryToCategory)

	res, err := entity.GetEntryByID(t.db, "err#123")
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 2, len(res.Categories))
		assert.Equal(t.T(), 1, res.Categories[0].CategoryID)
		assert.Equal(t.T(), "cat1", res.Categories[0].Category.Name)
		assert.Equal(t.T(), 2, res.Categories[1].CategoryID)
		assert.Equal(t.T(), "cat2", res.Categories[1].Category.Name)
		assert.Equal(t.T(), "err#123", res.GUID)
	}
}

func (t *SuiteTest) Test_Command_GetListBlocks() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)
	entry := []entity.Entry{{GUID: "err#123", Provider: provider}, {GUID: "err#321", Provider: provider}}
	t.db.Create(&entry)
	category := []entity.Category{{Name: "cat1", Provider: provider}, {Name: "cat2", Provider: provider}}
	t.db.Create(&category)
	entryToCategory := []entity.EntryToCategory{{EntryID: entry[0].GUID, CategoryID: category[0].ID}, {EntryID: entry[0].GUID, CategoryID: category[1].ID}, {EntryID: entry[1].GUID, CategoryID: category[1].ID}}
	t.db.Create(&entryToCategory)
	blockedCategory := []entity.BlockedCategory{{CategoryID: category[0].ID}, {CategoryID: category[1].ID}}
	t.db.Create(&blockedCategory)

	res, err := entity.GetListBlocks(t.db)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 2, len(res))
		assert.Equal(t.T(), category[0].ID, res[0].CategoryID)
		assert.Equal(t.T(), category[0].Name, res[0].Category.Name)
		assert.Equal(t.T(), provider.Lang, res[0].Category.Provider.Lang)
		assert.Equal(t.T(), category[1].ID, res[1].CategoryID)
		assert.Equal(t.T(), category[1].Name, res[1].Category.Name)
		assert.Equal(t.T(), provider.Lang, res[1].Category.Provider.Lang)
	}
}

func (t *SuiteTest) Test_Command_AddCategoryToBlock_DeleteCategoryFromBlock() {
	provider := entity.Provider{URL: "err.ee"}
	t.db.Create(&provider)
	entry := []entity.Entry{{GUID: "err#123", Provider: provider}, {GUID: "err#321", Provider: provider}}
	t.db.Create(&entry)
	category := []entity.Category{{Name: "cat1", Provider: provider}, {Name: "cat2", Provider: provider}}
	t.db.Create(&category)
	entryToCategory := []entity.EntryToCategory{{EntryID: entry[0].GUID, CategoryID: category[0].ID}, {EntryID: entry[0].GUID, CategoryID: category[1].ID}, {EntryID: entry[1].GUID, CategoryID: category[1].ID}}
	t.db.Create(&entryToCategory)

	res, err := entity.GetListBlocks(t.db)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 0, len(res))
	}
	err = entity.AddCategoryToBlock(t.db, category[0].ID)
	assert.NoError(t.T(), err)
	res, err = entity.GetListBlocks(t.db)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 1, len(res))
	}
	err = entity.AddCategoryToBlock(t.db, category[1].ID)
	assert.NoError(t.T(), err)
	res, err = entity.GetListBlocks(t.db)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 2, len(res))
	}
	err = entity.DeleteCategoryFromBlock(t.db, category[0].ID)
	assert.NoError(t.T(), err)
	res, err = entity.GetListBlocks(t.db)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 1, len(res))
		assert.Equal(t.T(), category[1].ID, res[0].CategoryID)
	}
}

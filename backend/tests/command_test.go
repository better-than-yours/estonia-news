package tests

import (
	"estonia-news/entity"

	"github.com/stretchr/testify/assert"
)

func (t *SuiteTest) Test_Command_GetEntryByID() {
	LoadFixtures(t)
	res, err := entity.GetEntryByID(t.ctx, "err#123")
	if assert.NoError(t.T(), err) {
		var categories []entity.Category
		_ = t.db.NewSelect().Model(&categories).Scan(t.ctx)
		assert.Equal(t.T(), 2, len(res.Categories))
		assert.Equal(t.T(), categories[0].ID, res.Categories[0].CategoryID)
		assert.Equal(t.T(), "cat1", res.Categories[0].Category.Name)
		assert.Equal(t.T(), categories[1].ID, res.Categories[1].CategoryID)
		assert.Equal(t.T(), "cat2", res.Categories[1].Category.Name)
		assert.Equal(t.T(), "err#123", res.ID)
	}
}

func (t *SuiteTest) Test_Command_GetListBlocks() {
	LoadFixtures(t)
	var categories []entity.Category
	_ = t.db.NewSelect().Model(&categories).Scan(t.ctx)
	var providers []entity.Provider
	_ = t.db.NewSelect().Model(&providers).Scan(t.ctx)
	res, err := entity.GetListBlocks(t.ctx)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 1, len(res))
		assert.Equal(t.T(), categories[0].ID, res[0].CategoryID)
		assert.Equal(t.T(), categories[0].Name, res[0].Category.Name)
		assert.Equal(t.T(), providers[0].Lang, res[0].Category.Provider.Lang)
	}
}

func (t *SuiteTest) Test_Command_AddCategoryToBlock_DeleteCategoryFromBlock() {
	LoadFixtures(t)
	var categories []entity.Category
	_ = t.db.NewSelect().Model(&categories).Scan(t.ctx)
	err := entity.AddCategoryToBlock(t.ctx, categories[0].ID)
	assert.NoError(t.T(), err)
	res, err := entity.GetListBlocks(t.ctx)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 1, len(res))
	}
	err = entity.AddCategoryToBlock(t.ctx, categories[1].ID)
	assert.NoError(t.T(), err)
	res, err = entity.GetListBlocks(t.ctx)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 2, len(res))
	}
	err = entity.DeleteCategoryFromBlock(t.ctx, categories[0].ID)
	assert.NoError(t.T(), err)
	res, err = entity.GetListBlocks(t.ctx)
	if assert.NoError(t.T(), err) {
		assert.Equal(t.T(), 1, len(res))
		assert.Equal(t.T(), categories[1].ID, res[0].CategoryID)
	}
}

package entity

import (
	"context"
	"estonia-news/config"
	"fmt"

	"github.com/uptrace/bun"
)

// GetEntryByID get entry info
func GetEntryByID(ctx context.Context, entryID string) (*Entry, error) {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	var entry Entry
	err := dbConnect.NewSelect().Model(&entry).Relation("Categories.Category").Where("id LIKE ?", fmt.Sprintf("%s%s%s", "%", entryID, "%")).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// AddCategoryToBlock add category to list blocks
func AddCategoryToBlock(ctx context.Context, categoryID int) error {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	_, err := dbConnect.NewInsert().Model(&BlockedCategory{
		CategoryID: categoryID,
	}).Ignore().Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// DeleteCategoryFromBlock delete category from list blocks
func DeleteCategoryFromBlock(ctx context.Context, categoryID int) error {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	_, err := dbConnect.NewDelete().Model(&BlockedCategory{}).Where("category_id = ?", categoryID).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// GetListBlocks return list blocks
func GetListBlocks(ctx context.Context) ([]BlockedCategory, error) {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	var blocks []BlockedCategory
	err := dbConnect.NewSelect().Model(&blocks).Relation("Category.Provider").Scan(ctx)
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

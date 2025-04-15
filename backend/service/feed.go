package service

import (
	"context"
	"database/sql"
	"errors"
	"estonia-news/config"
	"estonia-news/entity"
	"fmt"

	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"github.com/uptrace/bun"
)

// AddMissedCategories perform add missed categories
func AddMissedCategories(ctx context.Context, items []*gofeed.Item) (map[string]int, error) {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	provider := ctx.Value(config.CtxProviderKey).(*entity.Provider)
	categories := funk.FlatMap(items, func(v *gofeed.Item) []string {
		return v.Categories
	}).([]string)
	categories = funk.Uniq(categories).([]string)
	categoriesMap := make(map[string]int)
	for _, categoryName := range categories {
		var category entity.Category
		err := dbConnect.NewSelect().Model(&category).Where("name = ? AND provider_id = ?", categoryName, provider.ID).Limit(1).Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				category = entity.Category{
					Name:       categoryName,
					ProviderID: provider.ID,
				}
				_, err = dbConnect.NewInsert().Model(&category).On("CONFLICT (name, provider_id) DO NOTHING").Exec(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to add category '%s' for provider '%d': %v", categoryName, provider.ID, err)
				}
			} else {
				return nil, fmt.Errorf("failed to add category '%s' for provider '%d': %v", categoryName, provider.ID, err)
			}
		}
		categoriesMap[categoryName] = category.ID
	}
	return categoriesMap, nil
}

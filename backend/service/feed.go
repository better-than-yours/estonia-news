package service

import (
	"context"
	"estonia-news/config"
	"estonia-news/entity"

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
		category := entity.Category{
			Name:       categoryName,
			ProviderID: provider.ID,
		}
		_, err := dbConnect.NewInsert().Model(&category).On("CONFLICT (name, provider_id) DO NOTHING").Exec(ctx)
		if err != nil {
			return nil, err
		}
		if category.ID == 0 {
			err = dbConnect.NewSelect().Model(&category).Where("name = ? AND provider_id = ?", categoryName, provider.ID).Scan(ctx)
			if err != nil {
				return nil, err
			}
		}
		categoriesMap[categoryName] = category.ID
	}
	return categoriesMap, nil
}

package service

import (
	"estonia-news/config"
	"estonia-news/entity"

	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
)

// AddMissedCategories perform add missed categories
func AddMissedCategories(params *config.Params, items []*gofeed.Item) (map[string]int, error) {
	categories := funk.FlatMap(items, func(v *gofeed.Item) []string {
		return v.Categories
	}).([]string)
	categories = funk.Uniq(categories).([]string)
	categoriesMap := make(map[string]int)
	for _, categoryName := range categories {
		category := entity.Category{
			Name:     categoryName,
			Provider: params.Provider,
		}
		result := params.DB.Where(category).FirstOrCreate(&category)
		if result.Error != nil {
			return nil, result.Error
		}
		categoriesMap[categoryName] = category.ID
	}
	return categoriesMap, nil
}

package entity

import "gorm.io/gorm"

// GetEntryByID get entry info
func GetEntryByID(dbConnect *gorm.DB, entryID string) (*Entry, error) {
	var entry Entry
	result := dbConnect.Preload("Categories.Category").First(&entry, "guid = ?", entryID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &entry, nil
}

// AddCategoryToBlock add category to list blocks
func AddCategoryToBlock(dbConnect *gorm.DB, categoryID int) error {
	result := dbConnect.Create(&BlockedCategory{
		CategoryID: categoryID,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeleteCategoryFromBlock delete category from list blocks
func DeleteCategoryFromBlock(dbConnect *gorm.DB, categoryID int) error {
	result := dbConnect.Unscoped().Where("category_id = ?", categoryID).Delete(&BlockedCategory{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetListBlocks return list blocks
func GetListBlocks(dbConnect *gorm.DB) ([]BlockedCategory, error) {
	var blocks []BlockedCategory
	result := dbConnect.Preload("Category").Preload("Category.Provider").Find(&blocks)
	if result.Error != nil {
		return nil, result.Error
	}
	return blocks, nil
}

package entity

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Entry is a entry structure
type Entry struct {
	GUID        string `gorm:"primaryKey"`
	Link        string
	Title       string
	Description string
	MessageID   int
	Categories  []EntryToCategory `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ProviderID  int               `gorm:"index:provider_id_index;index:provider_id_published_index;index:provider_id_updated_at_index"`
	Provider    Provider          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UpdatedAt   time.Time         `gorm:"index:updated_at_index;index:provider_id_updated_at_index"`
	Published   time.Time         `gorm:"index:published_index;index:provider_id_published_index"`
}

// Provider is a provider structure
type Provider struct {
	ID                int `gorm:"primaryKey"`
	URL               string
	Lang              string
	BlockedCategories pq.StringArray `gorm:"type:text[]"`
	BlockedWords      pq.StringArray `gorm:"type:text[]"`
}

// Category is a category structure
type Category struct {
	ID         int      `gorm:"primaryKey"`
	Name       string   `gorm:"index:name_provider_unique,unique"`
	ProviderID int      `gorm:"index:name_provider_unique,unique"`
	Provider   Provider `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// EntryToCategory is a map entry and a category structures
type EntryToCategory struct {
	EntryID    string   `gorm:"primaryKey"`
	Entry      Entry    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CategoryID int      `gorm:"primaryKey"`
	Category   Category `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// UpsertCategory is function for upsert category
func UpsertCategory(db *gorm.DB, item *Category) *gorm.DB {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}, {Name: "provider_id"}},
		UpdateAll: true,
	}).Create(item)

}

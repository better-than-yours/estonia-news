// Package db handle work with db
package db

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Entry is a entry structure
type Entry struct {
	GUID        string `gorm:"uniqueIndex"`
	Link        string
	Title       string
	Description string
	ProviderID  int `gorm:"index:provider_id_index;index:provider_id_published_index;index:provider_id_updated_at_index"`
	MessageID   int
	Published   time.Time      `gorm:"index:published_index;index:provider_id_published_index"`
	Categories  pq.StringArray `gorm:"type:text[]"`
	Provider    Provider       `gorm:"index,constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UpdatedAt   time.Time      `gorm:"index:updated_at_index;index:provider_id_updated_at_index"`
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
	Provider   Provider `gorm:"index,constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// UpsertCategory is function for upsert category
func UpsertCategory(db *gorm.DB, item *Category) *gorm.DB {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}, {Name: "provider_id"}},
		UpdateAll: true,
	}).Create(item)

}

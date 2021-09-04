// Package db handle work with db
package db

import (
	"time"

	"github.com/lib/pq"
)

// Entry is a entry structure
type Entry struct {
	GUID        string `gorm:"index"`
	Link        string
	Title       string
	Description string
	Published   time.Time
	MessageID   int
	ProviderID  int
	Categories  pq.StringArray `gorm:"type:text[]"`
	Provider    Provider       `gorm:"index,constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UpdatedAt   time.Time      `gorm:"index"`
}

// Provider is a provider structure
type Provider struct {
	ID                uint `gorm:"primaryKey"`
	URL               string
	Lang              string
	BlockedCategories pq.StringArray `gorm:"type:text[]"`
	BlockedWords      pq.StringArray `gorm:"type:text[]"`
}

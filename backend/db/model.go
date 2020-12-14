// Package db handle work with db
package db

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Entry is a entry structure
type Entry struct {
	gorm.Model
	GUID        string
	Link        string
	Title       string
	Description string
	Published   time.Time
	MessageID   int
	ProviderID  int
	Categories  pq.StringArray `gorm:"type:text[]"`
	Provider    Provider       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Provider is a provider structure
type Provider struct {
	gorm.Model
	URL               string
	Lang              string
	BlockedCategories pq.StringArray `gorm:"type:text[]"`
}

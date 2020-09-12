// Package db handle work with db
package db

import (
	"time"

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
	Provider    Provider `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Provider is a provider structure
type Provider struct {
	gorm.Model
	URL  string
	Lang string
}

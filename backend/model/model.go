// Package model describes data models
package model

import (
	"time"
)

// Entry is a entry structure
type Entry struct {
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
	ID   int
	URL  string
	Lang string
}

package entity

import (
	"time"

	"github.com/uptrace/bun"
)

// Entry is a entry structure
type Entry struct {
	bun.BaseModel `bun:"table:entries,alias:e"`

	ID          string `bun:",pk"`
	Link        string
	Title       string
	Description string
	ImageURL    string
	Paywall     bool
	MessageID   int
	ProviderID  int
	UpdatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	PublishedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	Provider   *Provider          `bun:"rel:has-one,join:provider_id=id"`
	Categories []*EntryToCategory `bun:"rel:has-many,join:id=entry_id"`
}

// Provider is a provider structure
type Provider struct {
	bun.BaseModel `bun:"table:providers,alias:p"`

	ID             int `bun:",pk,autoincrement"`
	Name           string
	URL            string
	Lang           string
	BlockedWords   []string `bun:",array"`
	BlockedDomains []string `bun:",array"`
}

// Category is a category structure
type Category struct {
	bun.BaseModel `bun:"table:categories,alias:c"`

	ID         int       `bun:",pk,autoincrement"`
	Name       string    `bun:"unique:uniq_idx_categories"`
	ProviderID int       `bun:"unique:uniq_idx_categories"`
	Provider   *Provider `bun:"rel:has-one,join:provider_id=id"`
}

// EntryToCategory is a map entry and a category structures
type EntryToCategory struct {
	bun.BaseModel `bun:"table:entry_to_categories,alias:etc"`

	EntryID    string    `bun:",pk,unique:uniq_idx_entry_to_categories"`
	CategoryID int       `bun:",pk,unique:uniq_idx_entry_to_categories"`
	Category   *Category `bun:"rel:has-one,join:category_id=id"`
}

// BlockedCategory is list blocked categories
type BlockedCategory struct {
	bun.BaseModel `bun:"table:blocked_categories,alias:bc"`

	CategoryID int       `bun:",pk"`
	Category   *Category `bun:"rel:has-one,join:category_id=id"`
}

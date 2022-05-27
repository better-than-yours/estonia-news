package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"estonia-news/config"
	"estonia-news/db"
	"estonia-news/entity"

	"github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
)

type SuiteTest struct {
	suite.Suite
	db  *bun.DB
	ctx context.Context
}

func dropAllTables(t *SuiteTest) {
	rows, err := t.db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			panic(err)
		}
		dropTable(t, tableName)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
}

func dropTable(t *SuiteTest, tableName string) {
	_, err := t.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", pq.QuoteIdentifier(tableName)))
	if err != nil {
		panic(err)
	}
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteTest))
}

func (t *SuiteTest) SetupSuite() {
	t.db = db.Connect("127.0.0.1", "postgres", "postgres", "postgres")
	ctx := context.Background()
	t.ctx = context.WithValue(ctx, config.CtxDBKey, t.db)
	dropAllTables(t)
}

func (t *SuiteTest) SetupTest() {
	db.Migrate(t.ctx)
}

func (t *SuiteTest) TearDownTest() {
	dropAllTables(t)
}

func (t *SuiteTest) TearDownSuite() {
	dropAllTables(t)
}

func LoadFixtures(t *SuiteTest) {
	providers := []entity.Provider{{URL: "err.ee"}, {URL: "pm.ee"}}
	_, err := t.db.NewInsert().Model(&providers).Exec(t.ctx)
	if err != nil {
		fmt.Printf("%v", err)
	}
	entry := []entity.Entry{{ID: "err#123", ProviderID: providers[0].ID, PublishedAt: time.Now(), UpdatedAt: time.Now()}, {ID: "err#321", ProviderID: providers[0].ID, PublishedAt: time.Now(), UpdatedAt: time.Now()}}
	_, err = t.db.NewInsert().Model(&entry).Exec(t.ctx)
	if err != nil {
		fmt.Printf("%v", err)
	}
	category := []entity.Category{{Name: "cat1", ProviderID: providers[0].ID}, {Name: "cat2", ProviderID: providers[0].ID}}
	_, err = t.db.NewInsert().Model(&category).Exec(t.ctx)
	if err != nil {
		fmt.Printf("%v", err)
	}
	entryToCategory := []entity.EntryToCategory{{EntryID: entry[0].ID, CategoryID: category[0].ID}, {EntryID: entry[0].ID, CategoryID: category[1].ID}, {EntryID: entry[1].ID, CategoryID: category[1].ID}}
	_, err = t.db.NewInsert().Model(&entryToCategory).Exec(t.ctx)
	if err != nil {
		fmt.Printf("%v", err)
	}
	blockedCategory := []entity.BlockedCategory{{CategoryID: category[0].ID}}
	_, err = t.db.NewInsert().Model(&blockedCategory).Exec(t.ctx)
	if err != nil {
		fmt.Printf("%v", err)
	}
	t.ctx = context.WithValue(t.ctx, config.CtxProviderKey, &providers[0])
}

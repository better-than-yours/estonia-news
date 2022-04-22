package db

import (
	"fmt"

	"estonia-news/entity"
	"estonia-news/misc"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GetModels return list of models
func GetModels() []any {
	return []any{&entity.Provider{}, &entity.Category{}, &entity.Entry{}, &entity.EntryToCategory{}, &entity.BlockedCategory{}}
}

// Connect return db connection
func Connect(host, user, password, name string) *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("host=%s user=%s password=%s dbname=%s", host, user, password, name),
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		misc.Fatal("db_connect", "failed to connect database", err)
	}
	err = db.AutoMigrate(GetModels()...)
	if err != nil {
		misc.Fatal("db_migration", "db migration", err)
	}
	return db
}

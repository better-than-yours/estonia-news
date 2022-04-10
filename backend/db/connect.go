package db

import (
	"fmt"

	"estonia-news/entity"
	"estonia-news/misc"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect return db connection
func Connect(dbHost, dbUser, dbPassword, dbName string) *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("host=%s user=%s password=%s dbname=%s", dbHost, dbUser, dbPassword, dbName),
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		misc.TaskErrors.With(prometheus.Labels{"error": "db_connect"}).Inc()
		misc.PushMetrics()
		misc.L.Logf("FATAL failed to connect database, %v", err)
	}
	err = db.AutoMigrate(&entity.Provider{}, &entity.Category{}, &entity.Entry{}, &entity.EntryToCategory{})
	if err != nil {
		misc.TaskErrors.With(prometheus.Labels{"error": "db_migration"}).Inc()
		misc.PushMetrics()
		misc.L.Logf("FATAL db migration, %v", err)
	}
	return db
}

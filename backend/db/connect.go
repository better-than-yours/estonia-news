package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect - connection to a db
func Connect(dbHost, dbUser, dbPassword, dbName string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s", dbHost, dbUser, dbPassword, dbName)), &gorm.Config{})
	if err != nil {
		log.Fatalf("[ERROR] failed to connect database, %v", err)
	}
	err = db.AutoMigrate(&Entry{}, &Provider{})
	if err != nil {
		log.Fatalf("[ERROR] db migration, %v", err)
	}
	return db
}

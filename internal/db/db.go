package db

import (
	"errors"

	"github.com/acme-sky/workers/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Global variable but private
var db *gorm.DB = nil

// Init the database from a DSN string which must be a valid PostgreSQL dsn.
// Also, auto migrate all the models.
func InitDb(dsn string) (*gorm.DB, error) {
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err == nil {
		db.AutoMigrate(
			&models.AvailableFlight{},
			&models.Interest{},
			&models.Journey{},
			&models.Offer{},
			&models.Rent{},
			&models.Airline{},
			&models.Invoice{},
			&models.User{},
		)
	}

	return db, err
}

// Return the instance or error if the config is not laoded yet
func GetDb() (*gorm.DB, error) {
	if db == nil {
		return nil, errors.New("You must call `InitDb()` first.")
	}
	return db, nil
}

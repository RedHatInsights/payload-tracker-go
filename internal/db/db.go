package db

import (
	"fmt"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func DbConnect(cfg *config.TrackerConfig) {
	var (
		user     = cfg.DatabaseConfig.DBUser
		password = cfg.DatabaseConfig.DBPassword
		dbname   = cfg.DatabaseConfig.DBName
		host     = cfg.DatabaseConfig.DBHost
		port     = cfg.DatabaseConfig.DBPort
	)
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", user, password, dbname, host, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		l.Log.Fatal(err)
	}

	DB = db

	l.Log.Info("DB initialization complete")
}

package test

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
)

func WithDatabase() func() *gorm.DB {
	var db *gorm.DB

	BeforeEach(func() {
		var err error
		cfg := config.Get()
		var (
			user     = cfg.DatabaseConfig.DBUser
			password = cfg.DatabaseConfig.DBPassword
			dbname   = cfg.DatabaseConfig.DBName
			host     = cfg.DatabaseConfig.DBHost
			port     = cfg.DatabaseConfig.DBPort
		)

		dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", user, password, dbname, host, port)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if sqlConnection, err := db.DB(); err != nil {
			sqlConnection.Close()
		}
	})

	return func() *gorm.DB {
		return db
	}
}

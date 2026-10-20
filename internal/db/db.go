package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	l "github.com/redhatinsights/payload-tracker-go/internal/logging"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// buildDSN assembles the Postgres connection string. When Clowder provisions
// an RDS CA bundle, the connection is upgraded to sslmode=verify-full with
// that CA passed as sslrootcert so the server certificate and hostname are
// actually verified, rather than only encrypting the connection.
func buildDSN(cfg *config.TrackerConfig) (string, error) {
	var (
		user     = cfg.DatabaseConfig.DBUser
		password = cfg.DatabaseConfig.DBPassword
		dbname   = cfg.DatabaseConfig.DBName
		host     = cfg.DatabaseConfig.DBHost
		port     = cfg.DatabaseConfig.DBPort
		sslmode  = "disable"
	)

	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s", user, password, dbname, host, port, sslmode)

	if cfg.DatabaseConfig.RDSCa != "" {
		if _, err := os.Stat(cfg.DatabaseConfig.RDSCa); err != nil {
			return "", fmt.Errorf("RDS CA file %q is not accessible: %w", cfg.DatabaseConfig.RDSCa, err)
		}

		dsn = fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=verify-full sslrootcert=%s", user, password, dbname, host, port, cfg.DatabaseConfig.RDSCa)
	}

	return dsn, nil
}

func DbConnect(cfg *config.TrackerConfig) {
	dsn, err := buildDSN(cfg)
	if err != nil {
		l.Log.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		l.Log.Fatal(err)
	}

	DB = db

	l.Log.Info("DB initialization complete")
}

func DbSqlConnect(cfg *config.TrackerConfig) (*sql.DB, error) {
	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	return sql.Open("postgres", dsn)
}

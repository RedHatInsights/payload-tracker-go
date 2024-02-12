package main

import (
	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/logging"

	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/sirupsen/logrus"
)

func main() {
	logging.InitLogger()

	cfg := config.Get()

	databaseConn, err  := db.DbSqlConnect(cfg)
    if err != nil {
        panic(err)
    }

    err = performDbMigration(databaseConn, logging.Log, "file:///db/migrations", "up")
    if err != nil {
        panic(err)
    }
}

type loggerWrapper struct {
    *logrus.Logger
}

func (lw loggerWrapper) Verbose() bool {
	return true
}

func (lw loggerWrapper) Printf(format string, v ...interface{}) {
	lw.Infof(format, v...)
}

func performDbMigration(databaseConn *sql.DB, log *logrus.Logger, pathToMigrationFiles string, direction string) error {

	log.Info("Starting Payload-Tracker service DB migration")

	driver, err := postgres.WithInstance(databaseConn, &postgres.Config{})
	if err != nil {
		log.Error("Unable to get postgres driver from database connection", "error", err)
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(pathToMigrationFiles, "postgres", driver)
	if err != nil {
		log.Error("Unable to intialize database migration util", "error", err)
		return err
	}

	m.Log = loggerWrapper{log}

	if direction == "up" {
		err = m.Up()
	} else if direction == "down" {
		err = m.Steps(-1)
	} else {
		return errors.New("Invalid operation")
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("DB migration resulted in no changes")
	} else if err != nil {
		log.Error("DB migration resulted in an error", "error", err)
		return err
	}

	return nil
}

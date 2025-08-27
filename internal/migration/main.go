package main

import (
	"database/sql"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/sirupsen/logrus"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/logging"
)

func main() {

	logging.InitLogger()

	cfg := config.Get()

	databaseConn, err := db.DbSqlConnect(cfg)
	if err != nil {
		logging.Log.Fatal("Unable to intialize database connection: ", err)
	}

	var rootCmd = &cobra.Command{
		Use: "pt-migrate",
	}

	var upCmd = &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade to a later version",
		RunE: func(cmd *cobra.Command, args []string) error {
			performDbMigration(databaseConn, logging.Log, "file://migrations", "up")
			return nil
		},
	}

	var downCmd = &cobra.Command{
		Use:   "downgrade",
		Short: "Revert to a previous version",
		RunE: func(cmd *cobra.Command, args []string) error {
			performDbMigration(databaseConn, logging.Log, "file://migrations", "down")
			return nil
		},
	}

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
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
		log.Error("Unable to get postgres driver from database connection: ", err)
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(pathToMigrationFiles, "postgres", driver)
	if err != nil {
		log.Error("Unable to intialize database migration util: ", err)
		return err
	}

	m.Log = loggerWrapper{log}

	switch direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	default:
		return errors.New("Invalid operation")
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("DB migration resulted in no changes")
	} else if err != nil {
		log.Error("DB migration resulted in an error: ", err)
		return err
	}

	return nil
}

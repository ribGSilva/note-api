package schema

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ribgsilva/note-api/persistence/v1/schema"
	"github.com/ribgsilva/note-api/platform/env"
	"github.com/ribgsilva/note-api/sys"
	"go.uber.org/zap"
)

func ListCommands() {
	println("Schema Commands")
	println("\tcreate\t\t\t- Creates the schema")
	println("\tdelete\t\t\t- Deletes the schema")
	println("\thelp\t\t\t- Print the commands available")
}

func Run(options []string) {
	if len(options) == 0 {
		ListCommands()
		return
	}
	// empty logger
	log := zap.NewNop().Sugar()
	if err := initVars(log); err != nil {
		println("error:", err)
	}
	defer func() {
		if err := sys.R.Database.Close(); err != nil {
			log.Errorf("could not close db conn gracefully: %s", err)
		}
	}()
	switch options[0] {
	case "create":
		println("creating schema")
		if err := schema.Create(context.Background()); err != nil {
			println("failed to create schema:", err.Error())
		} else {
			println("created schema")
		}
	case "delete":
		println("deleting schema")
		if err := schema.Drop(context.Background()); err != nil {
			println("failed to delete schema:", err.Error())
		} else {
			println("deleted schema")
		}
	case "help":
		fallthrough
	default:
		ListCommands()
	}
}

func initVars(log *zap.SugaredLogger) error {
	sys.Configs.Database.ConnectionURL = env.OrDefault(log, "DATABASE_CONNECTION_URL", "root:admin@localhost:3306/note")
	sys.Configs.Database.PingTimeout = env.DurationDefault(log, "DATABASE_PING_TIMEOUT", "2s")
	sys.Configs.Database.OperationTimeout = env.DurationDefault(log, "DATABASE_OPERATION_TIMEOUT", "5s")

	// logger
	sys.R.Log = log

	// mysql
	var db *sql.DB
	if err := func() error {
		mysqlDb, err := sql.Open("mysql", sys.Configs.Database.ConnectionURL)
		if err != nil {
			return fmt.Errorf("error to connecto to database: %w", err)
		}
		dbCtx, dbCancel := context.WithTimeout(context.Background(), sys.Configs.Database.PingTimeout)
		defer dbCancel()
		if err := mysqlDb.PingContext(dbCtx); err != nil {
			return fmt.Errorf("could not connect to database: %w", err)
		}
		db = mysqlDb
		return nil
	}(); err != nil {
		return err
	}
	sys.R.Database = db
	return nil
}

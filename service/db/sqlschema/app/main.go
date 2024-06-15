package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/peterbourgon/ff"

	"github.com/ml8/tinyr/service/db"
	"github.com/ml8/tinyr/service/db/sqlschema/migrate"
	"github.com/ml8/tinyr/service/util"
)

var (
	logger *slog.Logger
	fs     = flag.NewFlagSet("migrate", flag.ExitOnError)

	// schema flags
	_         = fs.String("config", "", "config file")
	schemaDir = fs.String("schema_dir", "./", "location of cql schemas")
	dryRun    = fs.Bool("dry_run", false, "if true, output operations without affecting database")
	basename  = fs.String("basename", "schema", "base filename, without cql extension")

	// sql flags
	driver  = fs.String("driver", "mysql", "Database driver")
	connStr = fs.String("connStr", "", "Connection string")
)

func main() {
	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("TINYR"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser))

	logger = slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}))
	logger.Info("flags", "schema_dir", *schemaDir, "basename", *basename, "dry_run", *dryRun, "connStr", *connStr)

	util.OkOrDie(db.RunMigration(
		db.MigrationArgs{
			Logger:    logger,
			SchemaDir: *schemaDir,
			Basename:  *basename,
			DryRun:    *dryRun,
			Extension: "sql",
			Migrator: &migrate.SQLMigrator{
				Config: db.SQLConfig{
					Driver:     *driver,
					ConnString: *connStr,
				},
				Logger: logger,
			},
		}))
}

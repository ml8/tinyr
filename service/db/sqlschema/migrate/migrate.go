package migrate

import (
	"database/sql"
	"log/slog"
	"strings"

	"github.com/ml8/tinyr/service/db"
)

type SQLMigrator struct {
	Config db.SQLConfig
	Logger *slog.Logger
	db     *sql.DB
}

func (c *SQLMigrator) InitDB() (err error) {
	c.db, err = db.OpenSQLDB(c.Config)
	c.Logger.Info("opened database", "error", err)
	return
}

func (c *SQLMigrator) Complete() {
	c.db.Close()
	c.Logger.Info("closed database")
}

func (c *SQLMigrator) ApplySchema(s db.Schema, dry_run bool) (err error) {
	for i, next := range strings.Split(s.Schema, ";") {
		next = strings.TrimSpace(next)
		if next == "" {
			continue
		}
		next += ";"
		if dry_run {
			c.Logger.Info("query", "idx", i, "query", next)
		} else {
			_, err = c.db.Exec(next)
		}
		if err != nil {
			break
		}
	}
	return
}

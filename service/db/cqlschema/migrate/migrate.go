package migrate

import (
	"log/slog"
	"strings"

	"github.com/gocql/gocql"
	"github.com/ml8/tinyr/service/db"
)

type CQLMigrator struct {
	Hosts   []string
	Logger  *slog.Logger
	session *gocql.Session
}

func (c *CQLMigrator) InitDB() (err error) {
	cluster := gocql.NewCluster(c.Hosts...)
	c.session, err = cluster.CreateSession()
	c.Logger.Info("opened session", "error", err)
	return
}

func (c *CQLMigrator) Complete() {
	c.session.Close()
	c.Logger.Info("closed session")
}

func (c *CQLMigrator) ApplySchema(s db.Schema, dry_run bool) (err error) {
	for i, next := range strings.Split(s.Schema, ";") {
		next = strings.TrimSpace(next)
		if next == "" {
			continue
		}
		next += ";"
		if dry_run {
			c.Logger.Info("query", "idx", i, "query", next)
		} else {
			err = c.session.Query(next).Exec()
		}
		if err != nil {
			break
		}
	}
	return
}

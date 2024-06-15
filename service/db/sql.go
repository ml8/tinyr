package db

import (
	"context"
	"database/sql"
	"slices"

	_ "github.com/go-sql-driver/mysql"

	"github.com/ml8/tinyr/service/healthz"
	"github.com/ml8/tinyr/service/util"
)

var (
	knownDrivers = []string{"mysql"}
)

const (
	getShortQ    = "SELECT short_url, long_url, owner_id FROM shorts WHERE short_url=?"
	insertShortQ = "REPLACE INTO shorts (short_url, long_url, owner_id) VALUES (?, ?, ?)"
	deleteShortQ = "DELETE FROM shorts WHERE short_url=?"

	getUserQ    = "SELECT user_id, email, name FROM users WHERE user_id=?"
	insertUserQ = "REPLACE INTO users (user_id, email, name) VALUES (?, ?, ?)"
	deleteUserQ = "DELETE FROM users WHERE user_id=?"
)

type SQLConfig struct {
	Driver     string // mysql
	ConnString string // Connection string
}

type sqlStore struct {
	db *sql.DB
}

func (s sqlStore) Healthz(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

type sqlUserStore struct {
	sqlStore
}

type sqlShortStore struct {
	sqlStore
}

func OpenSQLDB(config SQLConfig) (db *sql.DB, err error) {
	if idx := slices.Index(knownDrivers, config.Driver); idx == -1 {
		logger.Error("unkown db driver", "driver", config.Driver, "allowed", knownDrivers)
		panic("unknown db driver")
	}
	db, err = sql.Open(config.Driver, config.ConnString)
	return
}

func NewSQLDB(config SQLConfig) Interface {
	db, err := OpenSQLDB(config)
	util.OkOrDie(err)
	s := sqlStore{db}
	healthz.Register(s)
	return container{
		s: &sqlShortStore{s},
		u: &sqlUserStore{s},
	}
}

func (s *sqlShortStore) Put(data ShortData) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var prev ShortData
	ok := true
	if err = tx.QueryRowContext(ctx, getShortQ, data.Short).Scan(&prev.Short, &prev.Long, &prev.Owner); err != nil {
		if err == sql.ErrNoRows {
			logger.Info("new row", "short", data.Short)
			ok = false
		} else {
			return err
		}
	}
	if ok && prev.Owner != data.Owner {
		logger.Info("not owned", "short", data.Short, "owner", prev.Owner, "new owner", data.Owner)
		return util.PermissionDeniedError
	}
	_, err = tx.ExecContext(ctx, insertShortQ, data.Short, data.Long, data.Owner)
	if err != nil {
		logger.Warn("failed to replace", "short", data.Short, "err", err)
		return err
	}
	err = tx.Commit()
	logger.Info("inserted", "short", data.Short, "err", err)
	return err
}

func (s *sqlShortStore) Get(short string) (data ShortData, err error) {
	err = s.db.QueryRow(getShortQ, short).Scan(&data.Short, &data.Long, &data.Owner)
	return
}

func (s *sqlShortStore) Delete(data ShortData) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var prev ShortData
	ok := true
	if err = tx.QueryRowContext(ctx, getShortQ, data.Short).Scan(&prev.Short, &prev.Long, &prev.Owner); err != nil {
		if err == sql.ErrNoRows {
			logger.Info("new row", "short", data.Short)
			ok = false
		} else {
			return err
		}
	}
	if ok && prev.Owner != data.Owner {
		logger.Info("not owned", "short", data.Short, "owner", prev.Owner, "new owner", data.Owner)
		return util.PermissionDeniedError
	}
	_, err = tx.ExecContext(ctx, deleteShortQ, data.Short)
	if err != nil {
		logger.Warn("failed to delete", "short", data.Short, "err", err)
		return err
	}
	err = tx.Commit()
	logger.Info("deleted", "short", data.Short, "err", err)
	return err
}

func (s *sqlShortStore) List(start string, end string) (ListResults, error) {
	panic("not implemented") // TODO: Implement
}

func (s *sqlUserStore) LookupOrCreate(queryUser UserData) (user UserData) {
	queryUser.Id = util.Hash(queryUser.Email)
	user = queryUser
	_, err := s.db.Exec(insertUserQ, user.Id, user.Email, user.Name)
	util.OkOrDie(err)
	return
}

func (s *sqlUserStore) Get(id uint64) (user UserData, err error) {
	err = s.db.QueryRow(getUserQ, id).Scan(&user.Id, &user.Email, &user.Name)
	return
}

func (s *sqlUserStore) Delete(id uint64) (err error) {
	_, err = s.db.Exec(deleteUserQ, id)
	return
}

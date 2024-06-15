package db

import (
	"context"
	"fmt"

	"github.com/gocql/gocql"
	gocqlx "github.com/scylladb/gocqlx/v2"
	"github.com/scylladb/gocqlx/v2/table"

	schema "github.com/ml8/tinyr/service/db/cqlschema"
	"github.com/ml8/tinyr/service/healthz"
	"github.com/ml8/tinyr/service/util"
)

func (u UserData) ToUsersStruct() schema.UsersStruct {
	return schema.UsersStruct{
		Email: u.Email,
		Name:  u.Name,
		Uid:   int64(u.Id),
	}
}

func ToUserData(u schema.UsersStruct) UserData {
	return UserData{
		Email: u.Email,
		Name:  u.Name,
		Id:    uint64(u.Uid),
	}
}

func (s ShortData) ToShortStruct() schema.ShortStruct {
	return schema.ShortStruct{
		Long:  s.Long,
		Short: s.Short,
		Owner: int64(s.Owner),
	}
}

func ToShortData(s schema.ShortStruct) ShortData {
	return ShortData{
		Short: s.Short,
		Long:  s.Long,
		Owner: uint64(s.Owner),
	}
}

type cqlDB struct {
	session  gocqlx.Session
	keyspace string
}

func (c *cqlDB) Healthz(ctx context.Context) error {
	return c.session.AwaitSchemaAgreement(ctx)
}

type CQLConfig struct {
	Hosts    []string
	Keyspace string
}

type cqlUserStore struct {
	cqlDB
	tbl *table.Table
}

type cqlShortStore struct {
	cqlDB
	tbl *table.Table
}

func NewCQLDB(config CQLConfig) Interface {
	db, err := cqlConnect(config)
	util.OkOrDie(err)
	// For health checking: session will attempt to heal.
	healthz.Register(&db)
	return container{
		s: &cqlShortStore{db, schema.Short},
		u: &cqlUserStore{db, schema.Users},
	}
}

func cqlConnect(config CQLConfig) (db cqlDB, err error) {
	cluster := gocql.NewCluster(config.Hosts...)
	cluster.Keyspace = config.Keyspace
	session, err := gocqlx.WrapSession(cluster.CreateSession())
	db = cqlDB{session: session, keyspace: config.Keyspace}
	return
}

func (c *cqlUserStore) LookupOrCreate(queryUser UserData) (user UserData) {
	queryUser.Id = util.Hash(queryUser.Email)
	u := queryUser.ToUsersStruct()
	s, n := c.tbl.Insert()
	s += "IF NOT EXISTS"
	q := c.session.Query(s, n).BindStruct(u)
	err := q.ExecRelease()
	util.OkOrDie(err)
	return queryUser
}

func (c *cqlUserStore) Get(id uint64) (user UserData, err error) {
	u := schema.UsersStruct{
		Uid: int64(id),
	}
	q := c.session.Query(c.tbl.Get()).BindStruct(u)
	err = q.GetRelease(&u)
	user = ToUserData(u)
	return
}

func (c *cqlUserStore) Delete(id uint64) (err error) {
	u := schema.UsersStruct{
		Uid: int64(id),
	}
	q := c.session.Query(c.tbl.Delete()).BindStruct(u)
	err = q.ExecRelease()
	return
}

func (c *cqlShortStore) Put(data ShortData) (err error) {
	d := data.ToShortStruct()
	s, n := c.tbl.Update("long", "owner")
	// Conditional mutation; requires coordination.
	s += fmt.Sprintf("IF owner = %v", data.Owner)
	q := c.session.Query(s, n).BindStruct(d)
	var applied bool
	applied, err = q.ExecCASRelease()
	if applied {
		logger.Info("updated", "key", data.Short, "owner", data.Owner)
	} else {
		logger.Info("new insert", "key", data.Short, "owner", data.Owner)
		s, n = c.tbl.Insert()
		// See note below. We use IF NOT EXISTS to prevent this user from
		// overwriting another user's racing insert.
		s += "IF NOT EXISTS"
		q = c.session.Query(s, n).BindStruct(d)
		applied, err = q.ExecCASRelease()
		if !applied && err == nil {
			// This is a small race condition. If the same owner retried the
			// insert we should either allow the insert to be idempotent. If another
			// owner did an insert, we should give permission denied.
			//
			// Since we cannot distinguish between the two cases, we return a blanket
			// error. TODO: issue read query for DB state for correct return value.
			err = util.InternalError
		}
	}
	return
}

func (c *cqlShortStore) Get(short string) (data ShortData, err error) {
	d := schema.ShortStruct{
		Short: short,
	}
	q := c.session.Query(c.tbl.Get()).BindStruct(d)
	err = q.GetRelease(&d)
	data = ToShortData(d)
	return
}

func (c *cqlShortStore) Delete(entry ShortData) (err error) {
	d := schema.ShortStruct{
		Short: entry.Short,
	}
	s, n := c.tbl.Delete()
	s += fmt.Sprintf("IF owner=%v", entry.Owner)
	q := c.session.Query(s, n).BindStruct(d)
	applied, err := q.ExecCASRelease()
	if !applied && err == nil {
		err = util.PermissionDeniedError
	}
	return
}

func (c *cqlShortStore) List(start string, end string) (ListResults, error) {
	panic("not implemented") // TODO: Implement
}

package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/ml8/tinyr/service/util"
)

const (
	InMemory = iota
	Pebble
	CQL
	SQL
)

var (
	logger *slog.Logger
)

type Config struct {
	Type   int
	Pebble PebbleConfig
	CQL    CQLConfig
	SQL    SQLConfig
	Logger *slog.Logger
}

type ShortData struct {
	Short string `json:"Short"`
	Long  string `json:"Long"`
	Owner uint64 `json:"Owner"`
}

type ShortStore interface {
	Put(data ShortData) error
	Get(short string) (ShortData, error)
	Delete(data ShortData) error
	List(start, end string) (ListResults, error)
}

type UserData struct {
	Email string `json:"Email"`
	Name  string `json:"Name"`
	Id    uint64 `json:"Id"`
}

type UserStore interface {
	LookupOrCreate(queryUser UserData) (user UserData)
	Get(id uint64) (user UserData, err error)
	Delete(id uint64) (err error)
}

type Interface interface {
	Shorts() ShortStore
	Users() UserStore
}

func (s ShortData) Encode() (b []byte, err error) {
	return json.Marshal(s)
}

func Decode(b []byte) (d ShortData, err error) {
	d = ShortData{}
	err = json.Unmarshal(b, &d)
	return
}

func New(config Config) Interface {
	logger = config.Logger
	logger.Info("database config", "config", config)
	switch config.Type {
	case InMemory:
		return NewInMemory()
	case Pebble:
		return NewPebble(config.Pebble)
	case CQL:
		return NewCQLDB(config.CQL)
	case SQL:
		return NewSQLDB(config.SQL)
	default:
		logger.Error("Invalid type", "type", config.Type)
		panic(errors.New(fmt.Sprintf("%v is not a valid database type", config.Type)))
	}
}

type ListResults struct {
	Matching []ShortData
}

type container struct {
	s ShortStore
	u UserStore
}

type ephemeralShortStore struct {
	sync.RWMutex
	sdb map[string]ShortData
}

type ephemeralUserStore struct {
	sync.RWMutex
	udb map[uint64]UserData
}

func NewInMemory() Interface {
	return container{
		s: &ephemeralShortStore{sync.RWMutex{}, make(map[string]ShortData)},
		u: &ephemeralUserStore{sync.RWMutex{}, make(map[uint64]UserData)}}
}

func (c container) Shorts() ShortStore {
	return c.s
}

func (c container) Users() UserStore {
	return c.u
}

func (db *ephemeralShortStore) Get(key string) (entry ShortData, err error) {
	db.RLock()
	defer db.RUnlock()
	var ok bool
	if entry, ok = db.sdb[key]; !ok {
		err = util.NoSuchKeyError(key)
		return
	}
	return
}

func (db *ephemeralShortStore) Put(entry ShortData) (err error) {
	db.Lock() // Do not interleave writes.
	defer db.Unlock()
	prev, ok := db.sdb[entry.Short]
	if ok && prev.Owner != entry.Owner {
		err = util.PermissionDeniedError
		return
	}
	db.sdb[entry.Short] = entry
	return
}

func (db *ephemeralShortStore) Delete(entry ShortData) (err error) {
	db.Lock() // Do not interleave writes.
	defer db.Unlock()
	prev, ok := db.sdb[entry.Short]
	if ok && prev.Owner != entry.Owner {
		err = util.PermissionDeniedError
		return
	}
	delete(db.sdb, entry.Short)
	return
}

func (db *ephemeralShortStore) List(start, end string) (results ListResults, err error) {
	db.RLock()
	defer db.RUnlock()
	results = ListResults{}
	for k, v := range db.sdb {
		if k >= start && k <= end {
			results.Matching = append(results.Matching, v)
		}
	}
	return
}

func (db *ephemeralUserStore) LookupOrCreate(queryUser UserData) (user UserData) {
	db.Lock()
	defer db.Unlock()
	// We lookup users by email.
	hash := util.Hash(queryUser.Email)
	if user, ok := db.udb[hash]; ok {
		return user
	}
	user = UserData{Email: queryUser.Email, Name: queryUser.Name, Id: hash}
	db.udb[hash] = user
	return
}

func (db *ephemeralUserStore) Get(id uint64) (user UserData, err error) {
	db.RLock()
	defer db.RUnlock()
	var ok bool
	if user, ok = db.udb[id]; ok {
		return
	}
	err = util.NoSuchKeyError(fmt.Sprintf("%d", id))
	return
}

func (db *ephemeralUserStore) Delete(id uint64) (err error) {
	db.Lock()
	defer db.Unlock()
	delete(db.udb, id)
	return
}

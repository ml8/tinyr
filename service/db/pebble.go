package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/ml8/tinyr/service/util"
)

type pebDB struct {
	db *pebble.DB
}

type pebbleShortStore struct {
	sync.Mutex // Do not interleave writes; put is not atomic.
	keyspace   string
	db         *pebble.DB
}

type pebbleUserStore struct {
	keyspace string
	db       *pebble.DB
}

const (
	shortKeyspace = "s"
	userKeyspace  = "u"
)

type PebbleConfig struct {
	Path string
}

func gobEncode[T any](e T) (encoded []byte) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	util.OkOrDie(enc.Encode(e))
	encoded = buf.Bytes()
	return
}

func gobDecode[T any](encoded []byte) (e T) {
	buf := bytes.NewBuffer(encoded)
	dec := gob.NewDecoder(buf)
	util.OkOrDie(dec.Decode(&e))
	return
}

func NewPebble(config PebbleConfig) Interface {
	db, err := pebble.Open(config.Path, &pebble.Options{})
	util.OkOrDie(err)
	gob.Register(ShortData{})
	gob.Register(UserData{})
	return container{
		s: &pebbleShortStore{Mutex: sync.Mutex{}, keyspace: shortKeyspace, db: db},
		u: &pebbleUserStore{keyspace: userKeyspace, db: db},
	}
}

func (p *pebbleShortStore) Get(key string) (entry ShortData, err error) {
	k := []byte(p.keyspace + key)
	val, closer, err := p.db.Get(k)
	if closer != nil {
		defer closer.Close()
	}
	if err != nil && errors.Is(err, pebble.ErrNotFound) {
		err = util.NoSuchKeyError(key)
		return
	}
	entry = gobDecode[ShortData](val)
	return
}

func (p *pebbleShortStore) Put(entry ShortData) (err error) {
	p.Lock()
	defer p.Unlock()
	var prev ShortData
	prev, err = p.Get(entry.Short)
	if _, ok := err.(util.NoSuchKeyError); !ok && err != nil {
		return
	}
	if prev.Short != "" && prev.Owner != entry.Owner {
		err = util.PermissionDeniedError
		return
	}
	err = p.db.Set([]byte(p.keyspace+entry.Short), gobEncode(entry), pebble.Sync)
	return
}

func (p *pebbleShortStore) Delete(entry ShortData) (err error) {
	p.Lock()
	defer p.Unlock()
	var prev ShortData
	prev, err = p.Get(entry.Short)
	if _, ok := err.(util.NoSuchKeyError); !ok && err != nil {
		return
	}
	if prev.Short != "" && prev.Owner != entry.Owner {
		err = util.PermissionDeniedError
		return
	}
	err = p.db.Delete([]byte(p.keyspace+entry.Short), pebble.Sync)
	return
}

func (p *pebbleShortStore) List(start, end string) (results ListResults, err error) {
	lb := []byte(p.keyspace)
	ub := []byte(string(p.keyspace[0] + 1))
	if start != "" {
		lb = []byte(p.keyspace + start)
	}
	if end != "" {
		ub = []byte(p.keyspace + end)
	}

	it, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: lb,
		UpperBound: ub,
	})
	if err != nil {
		return
	}

	for it.First(); it.Valid(); it.Next() {
		results.Matching = append(results.Matching, gobDecode[ShortData](it.Value()))
	}
	util.OkOrDie(it.Close())
	return
}

func (p *pebbleUserStore) keyFromEmail(email string) string {
	return p.keyFromId(util.Hash(email))
}

func (p *pebbleUserStore) keyFromId(id uint64) string {
	return p.keyspace + fmt.Sprintf("%d", id)
}

func (p *pebbleUserStore) LookupOrCreate(queryUser UserData) (user UserData) {
	var err error
	logger.Info("Query user", "user", queryUser)
	user, err = p.Get(util.Hash(queryUser.Email))
	if err != nil {
		return
	}
	if _, ok := err.(util.NoSuchKeyError); !ok {
		panic(err)
	}
	user = queryUser
	user.Id = util.Hash(queryUser.Email)
	err = p.db.Set([]byte(p.keyFromId(user.Id)), gobEncode(user), pebble.Sync)
	return
}

func (p *pebbleUserStore) Get(id uint64) (user UserData, err error) {
	k := []byte(p.keyFromId(id))
	val, closer, err := p.db.Get(k)
	if closer != nil {
		defer closer.Close()
	}
	if err != nil && errors.Is(err, pebble.ErrNotFound) {
		err = util.NoSuchKeyError(p.keyFromId(id))
		return
	}
	user = gobDecode[UserData](val)
	return
}

func (p *pebbleUserStore) Delete(id uint64) (err error) {
	err = p.db.Delete([]byte(p.keyFromId(id)), pebble.Sync)
	return
}

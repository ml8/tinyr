package db

import (
	"testing"

	"github.com/ml8/tinyr/service/util"
)

func TestPutGet(t *testing.T) {
	db := New(Config{Type: InMemory})
	db.Shorts().Put(ShortData{"miserable", "pigeon", 0})
	v, err := db.Shorts().Get("miserable")
	if err != nil {
		t.Errorf("Got error %v", err)
	} else if v.Long != "pigeon" {
		t.Errorf("Incorrect value %v", v.Long)
	}
}

func TestPutDeleteGet(t *testing.T) {
	db := New(Config{Type: InMemory})
	db.Shorts().Put(ShortData{"miserable", "pigeon", 0})
	err := db.Shorts().Delete("miserable")
	if err != nil {
		t.Errorf("Got error %v", err)
	}

	val, err := db.Shorts().Get("miserable")
	if err == nil {
		t.Errorf("Key should've been removed but got %v", val)
	} else if err != util.NoSuchKeyError("miserable") {
		t.Errorf("Incorrect error %v", err)
	}
}

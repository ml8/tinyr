package cache

import (
	"fmt"
	"testing"
)

func putN(cache KVCache[int], n int) {
	for i := 0; i < n; i++ {
		cache.Put(fmt.Sprintf("%d", i), i)
	}
}

func TestEvictLast(t *testing.T) {
	cache := New[int](5)
	putN(cache, 5)

	// should evict an entry.
	cache.Put("hello", 42)
	v, e := cache.Get(fmt.Sprintf("%d", 0))
	if e == nil {
		t.Errorf("Key 0 should've been evicted. Got %v", v)
	} else if _, ok := e.(NoSuchKeyError); !ok {
		t.Errorf("Error type should be NoSuchKeyError; got %v", e)
	}

	// should not evict an entry.
	cache.Put("hello", 36)
	v, e = cache.Get(fmt.Sprintf("%d", 1))
	if e != nil {
		t.Errorf("Key 1 shouldn't have been evicted. Got err %v", e)
	} else if v != 1 {
		t.Errorf("Incorrect value for key %v: %v", 1, v)
	}
}

func TestInvalidate(t *testing.T) {
	cache := New[int](5)

	putN(cache, 5)

	// oldest
	cache.Invalidate(fmt.Sprintf("%v", 0))
	v, e := cache.Get(fmt.Sprintf("%d", 0))
	if e == nil {
		t.Errorf("Key 0 should've been invalidated. Got %v", v)
	} else if _, ok := e.(NoSuchKeyError); !ok {
		t.Errorf("Error type should be NoSuchKeyError; got %v", e)
	}

	// newest
	cache.Invalidate(fmt.Sprintf("%v", 4))
	v, e = cache.Get(fmt.Sprintf("%d", 4))
	if e == nil {
		t.Errorf("Key 0 should've been invalidated. Got %v", v)
	} else if _, ok := e.(NoSuchKeyError); !ok {
		t.Errorf("Error type should be NoSuchKeyError; got %v", e)
	}
}

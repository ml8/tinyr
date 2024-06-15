package cache

import (
	"container/heap"
	"sync"
	"time"

	"github.com/ml8/tinyr/service/util"
)

type KVEntry[T any] struct {
	Key       string
	Value     T
	Timestamp int64
	entry     *qentry
}

type KVCache[T any] interface {
	Put(key string, value T) (previous T, err error)
	Get(key string) (value T, err error)
	Invalidate(key string) (value T, err error)
}

type qentry struct {
	value     uint64
	timestamp int64
	index     int // as an optimization we store the current index; this allows us to fix the heap in O(log n) vs. O(n)
}

type queue []*qentry

type cache[T any] struct {
	sync.Mutex
	size    int
	entries map[uint64]*KVEntry[T]
	pq      queue
}

// Implement sort.Interface for queue
func (pq queue) Len() int {
	return len(pq)
}
func (pq queue) Less(i, j int) bool {
	return pq[i].timestamp < pq[j].timestamp
}
func (pq queue) Swap(i, j int) {
	t := pq[i]
	pq[i] = pq[j]
	pq[j] = t
	pq[i].index = i
	pq[j].index = j
}

// Implement heap.Interface for queue
func (pq *queue) Push(x any) {
	x.(*qentry).index = len(*pq)
	*pq = append(*pq, x.(*qentry))
}
func (pq *queue) Pop() any {
	el := (*pq)[len(*pq)-1]
	*pq = (*pq)[0 : len(*pq)-1]
	return el
}

func New[T any](size int) KVCache[T] {
	return &cache[T]{
		size:    size,
		entries: make(map[uint64]*KVEntry[T]),
		pq:      make([]*qentry, 0, size)}
}

func (c *cache[T]) maybeEvict() {
	if len(c.entries) < c.size {
		return
	}
	el := heap.Pop(&c.pq).(*qentry)
	delete(c.entries, el.value)
}

func (c *cache[T]) access(key uint64, ts int64) {
	// Update timestamp for pq and fix heap
	e := c.entries[key]
	e.entry.timestamp = ts
	heap.Fix(&c.pq, e.entry.index)
}

func (c *cache[T]) remove(key uint64) {
	// remove entry from pq and map
	e := c.entries[key]
	heap.Remove(&c.pq, e.entry.index)
	delete(c.entries, key)
}

func (c *cache[T]) Put(key string, value T) (previous T, err error) {
	c.Lock()
	defer c.Unlock()
	h := util.Hash(key)
	ts := time.Now().UnixNano()
	if entry, ok := c.entries[h]; ok {
		// no eviction; replace value.
		previous = entry.Value
		entry.Timestamp = ts
		entry.Value = value
		c.access(h, ts)
		return
	}

	// new key.
	c.maybeEvict()
	n := &KVEntry[T]{
		Key:       key,
		Value:     value,
		Timestamp: ts,
		entry:     &qentry{h, ts, 0}}
	c.entries[h] = n
	heap.Push(&c.pq, n.entry)
	return
}

func (c *cache[T]) Get(key string) (value T, err error) {
	c.Lock()
	defer c.Unlock()
	h := util.Hash(key)
	if entry, ok := c.entries[h]; ok {
		value = entry.Value
		c.access(h, time.Now().UnixNano())
	} else {
		err = util.NoSuchKeyError(key)
	}
	return
}

func (c *cache[T]) Invalidate(key string) (value T, err error) {
	c.Lock()
	defer c.Unlock()
	h := util.Hash(key)
	if entry, ok := c.entries[h]; ok {
		value = entry.Value
		c.remove(h)
		return
	}
	return
}

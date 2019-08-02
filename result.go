package query

import (
	"math"
	"sync/atomic"
	"time"
)

// Result is the struct returned from regular queries.
type Result struct {
	resultCore
	data     []interface{}
	cacheKey []byte
	cachedAt time.Time
}

func newResult() *Result {
	return &Result{
		resultCore: newResultCore(),
		data:       make([]interface{}, 0, 1),
	}
}

func newCacheableResult(query Cacheable) *Result {
	return &Result{
		resultCore: newResultCore(),
		cacheKey:   query.CacheKey(),
		data:       make([]interface{}, 0, 1),
	}
}

// CacheKey is used to identify this result while cached
func (res *Result) CacheKey() []byte {
	return res.cacheKey
}

// CachedAt is used to identify at which point this result was cached
func (res *Result) CachedAt() time.Time {
	return res.cachedAt
}

//------Provide Data------//

// Set all the data of this result
func (res *Result) Set(data []interface{}) {
	res.data = data
}

// Add an entry to the data slice
func (res *Result) Add(data interface{}) {
	if len(res.data) == cap(res.data) {
		res.increaseCapacity()
	}
	res.data = append(res.data, data)
}

//------Fetch Data------//

// First returns the first value of the data slice
func (res *Result) First() interface{} {
	if len(res.data) <= 0 {
		return nil
	}
	return res.data[0]
}

// All returns the data slice
func (res *Result) All() []interface{} {
	return res.data
}

//------Internal------//

func (res *Result) increaseCapacity() {
	l := len(res.data)
	c := int(math.Ceil(float64(cap(res.data)) * 1.1))

	data := make([]interface{}, l, c)
	copy(data, res.data)
	res.data = data
}

func (res *Result) cached() {
	res.cachedAt = time.Now()
}

func (res *Result) isHandled() bool {
	return len(res.data) > 0 || atomic.LoadUint32(res.handled) == 1
}

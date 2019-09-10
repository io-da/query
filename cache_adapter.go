package query

import "time"

// CacheAdapter must be implemented for a type to qualify as a cache adapter.
type CacheAdapter interface {
	Set(qry Cacheable, res *Result, at time.Time) bool
	Get(qry Cacheable) *Result
	Expire(qry Cacheable)
	Shutdown()
}

package query

// CacheAdapter must be implemented for a type to qualify as a cache adapter.
type CacheAdapter interface {
	Set(qry Cacheable, res *Result) bool
	Get(qry Cacheable) *Result
	Expire(qry Cacheable)
	Shutdown()
}

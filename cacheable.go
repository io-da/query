package query

import "time"

// Cacheable is an interface used to allow queries to cache their results.
type Cacheable interface {
	CacheKey() []byte
	CacheDuration() time.Duration
}

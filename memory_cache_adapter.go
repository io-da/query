package query

import (
	"sync"
	"sync/atomic"
	"time"
)

// MemoryCacheAdapter is the struct used for memory caching purposes.
type MemoryCacheAdapter struct {
	sync.Mutex
	cachedResults map[string]*Result
	cleanerSignal chan bool
	shuttingDown  *uint32
	sleepTimer    *time.Timer
	sleepUntil    time.Time
}

// NewMemoryCacheAdapter initializes a new *MemoryCacheAdapter.
// This function will also initialize the respective cleaner routine.
func NewMemoryCacheAdapter() *MemoryCacheAdapter {
	ad := &MemoryCacheAdapter{
		cachedResults: make(map[string]*Result),
		cleanerSignal: make(chan bool, 1),
		shuttingDown:  new(uint32),
	}
	go ad.cleaner()
	return ad
}

// Set stores the cache value for the given query.
func (ad *MemoryCacheAdapter) Set(qry Cacheable, res *Result, at time.Time) bool {
	ad.Lock()
	ad.cachedResults[string(qry.CacheKey())] = res
	ad.updateSleepUntil(at.Add(qry.CacheDuration()))
	ad.clean()
	ad.Unlock()
	return true
}

// Get retrieves the cached result for the provided query.
func (ad *MemoryCacheAdapter) Get(qry Cacheable) *Result {
	ad.Lock()
	res, isCached := ad.cachedResults[string(qry.CacheKey())]
	ad.Unlock()
	if isCached {
		return res
	}
	return nil
}

// Expire can optionally be used to forcibly expire a query cache.
func (ad *MemoryCacheAdapter) Expire(qry Cacheable) {
	ck := string(qry.CacheKey())
	ad.Lock()
	if _, isCached := ad.cachedResults[ck]; isCached {
		delete(ad.cachedResults, ck)
	}
	ad.Unlock()
}

// Shutdown is used to stop the cleaner routine.
func (ad *MemoryCacheAdapter) Shutdown() {
	atomic.CompareAndSwapUint32(ad.shuttingDown, 0, 1)
	ad.clean()
}

//------Internal------//

func (ad *MemoryCacheAdapter) cleaner() {
	for atomic.LoadUint32(ad.shuttingDown) == 0 {
		ad.Lock()
		now := time.Now()
		ad.sleepUntil = time.Time{}
		for key, res := range ad.cachedResults {
			if !res.CachedAt().IsZero() && now.After(res.ExpiresAt()) {
				delete(ad.cachedResults, key)
				continue
			}
			ad.updateSleepUntil(res.ExpiresAt())
		}
		ad.updateSleepTimer(ad.determineSleepDuration())
		ad.Unlock()

		// allow the cleaner to be triggered either with timer or directly
		select {
		case <-ad.sleepTimer.C:
		case <-ad.cleanerSignal:
		}
	}
}

func (ad *MemoryCacheAdapter) clean() {
	select {
	case ad.cleanerSignal <- true:
	default:
	}
}

func (ad *MemoryCacheAdapter) updateSleepUntil(expiresAt time.Time) {
	if ad.sleepUntil.IsZero() || expiresAt.Before(ad.sleepUntil) {
		ad.sleepUntil = expiresAt
	}
}

func (ad *MemoryCacheAdapter) determineSleepDuration() time.Duration {
	if ad.sleepUntil.IsZero() || len(ad.cachedResults) <= 0 {
		return time.Hour
	}

	return ad.sleepUntil.Sub(time.Now())
}

func (ad *MemoryCacheAdapter) updateSleepTimer(d time.Duration) {
	if ad.sleepTimer == nil {
		ad.sleepTimer = time.NewTimer(d)
		return
	}
	ad.sleepTimer.Reset(d)
}

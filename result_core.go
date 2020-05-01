package query

import (
	"sync/atomic"
)

type resultCore struct {
	stopPropagation *uint32
	handled         *uint32
	fresh           *uint32
}

func newResultCore() resultCore {
	res := resultCore{
		stopPropagation: new(uint32),
		handled:         new(uint32),
		fresh:           new(uint32),
	}
	atomic.SwapUint32(res.fresh, 1)
	return res
}

//------Core------//

// Handled explicitly marks this result as handled (so no error is thrown if no data is found)
func (res *resultCore) Handled() {
	atomic.CompareAndSwapUint32(res.handled, 0, 1)
}

// Done explicitly marks this result as handled and final. Meaning it will not be propagated to other handlers.
func (res *resultCore) Done() {
	res.Handled()
	atomic.CompareAndSwapUint32(res.stopPropagation, 0, 1)
}

// IsFresh can be used to verify if this result is fresh.
func (res *resultCore) IsFresh() bool {
	return atomic.LoadUint32(res.fresh) == 1
}

// IsCached can be used to verify if this result was retrieved from cache.
func (res *resultCore) IsCached() bool {
	return atomic.LoadUint32(res.fresh) == 0
}

//------Internal------//

func (res *resultCore) propagationStopped() bool {
	return atomic.LoadUint32(res.stopPropagation) == 1
}

func (res *resultCore) isHandled() bool {
	return atomic.LoadUint32(res.handled) == 1
}

func (res *resultCore) loadedFromCache() {
	atomic.CompareAndSwapUint32(res.fresh, 1, 0)
}

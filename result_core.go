package query

import (
	"sync/atomic"
)

type resultCore struct {
	stopPropagation *uint32
	handled         *uint32
}

func newResultCore() resultCore {
	return resultCore{
		stopPropagation: new(uint32),
		handled:         new(uint32),
	}
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

//------Internal------//

func (res *resultCore) propagationStopped() bool {
	return atomic.LoadUint32(res.stopPropagation) == 1
}

func (res *resultCore) isHandled() bool {
	return atomic.LoadUint32(res.handled) == 1
}

package query

import "time"

// IteratorResult is the struct returned from iterator queries.
type IteratorResult struct {
	resultCore
	proxy     chan interface{}
	listening chan bool
}

func newIteratorResult(buffer int) *IteratorResult {
	return &IteratorResult{
		resultCore: newResultCore(),
		proxy:      make(chan interface{}, buffer),
		listening:  make(chan bool, 1),
	}
}

//------Provide Data------//

// Yield is used to provide values while they are being processed
func (res *IteratorResult) Yield(data interface{}) {
	res.Handled()
	res.proxy <- data
}

//------Fetch Data------//

// Iterate is used to process the values that are being yielded
func (res *IteratorResult) Iterate() <-chan interface{} {
	res.listening <- true
	return res.proxy
}

//------Internal------//

func (res *IteratorResult) waitListener(timeout time.Duration) bool {
	select {
	case <-res.listening:
		return true
	default:
		t := time.NewTimer(timeout)
		select {
		case <-res.listening:
			t.Stop()
			return true
		case <-t.C:
			return false
		}
	}
}

func (res *IteratorResult) close() {
	close(res.proxy)
}

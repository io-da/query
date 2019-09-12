package query

import (
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

const iteratorListenerTimeout = time.Second

// Bus is the only struct exported and required for the query bus usage.
// The Bus should be instantiated using the NewBus function.
type Bus struct {
	iteratorWorkerPoolSize int
	iteratorQueueBuffer    int
	iteratorResultBuffer   int
	initialized            *uint32
	shuttingDown           *uint32
	iteratorWorkers        *uint32
	handlers               []Handler
	iteratorHandlers       []IteratorHandler
	errorHandlers          []ErrorHandler
	cacheAdapters          []CacheAdapter
	iteratorQueryQueue     chan *pendingIteratorQuery
	closed                 chan bool
}

// NewBus instantiates the Bus struct.
// The Initialization of IteratorHandlers is performed separately (InitializeIteratorHandlers function) for dependency injection purposes.
func NewBus() *Bus {
	return &Bus{
		iteratorWorkerPoolSize: runtime.GOMAXPROCS(0),
		iteratorQueueBuffer:    100,
		iteratorResultBuffer:   0,
		initialized:            new(uint32),
		shuttingDown:           new(uint32),
		iteratorWorkers:        new(uint32),
		handlers:               make([]Handler, 0),
		iteratorHandlers:       make([]IteratorHandler, 0),
		errorHandlers:          make([]ErrorHandler, 0),
		cacheAdapters:          []CacheAdapter{NewMemoryCacheAdapter()},
		closed:                 make(chan bool),
	}
}

// Handlers for the regular queries.
func (bus *Bus) Handlers(hdls ...Handler) {
	bus.handlers = hdls
}

// ErrorHandlers may optionally be provided.
// They will receive any error thrown during the querying process.
func (bus *Bus) ErrorHandlers(hdls ...ErrorHandler) {
	bus.errorHandlers = hdls
}

// CacheAdapters may optionally be provided.
// They will be used instead of the default MemoryCacheAdapter.
func (bus *Bus) CacheAdapters(adps ...CacheAdapter) {
	for _, adp := range bus.cacheAdapters {
		adp.Shutdown()
	}
	bus.cacheAdapters = adps
}

// IteratorWorkerPoolSize may optionally be provided to tweak the iteratorWorker pool size for iterator query queue.
// It can only be adjusted *before* the bus is initialized.
// It defaults to the value returned by runtime.GOMAXPROCS(0).
func (bus *Bus) IteratorWorkerPoolSize(workerPoolSize int) {
	if !bus.isInitialized() {
		bus.iteratorWorkerPoolSize = workerPoolSize
	}
}

// IteratorQueueBuffer may optionally be provided to tweak the buffer size of the iterator query queue.
// This value may have high impact on performance depending on the use case.
// It can only be adjusted *before* the bus is initialized.
// It defaults to 100.
func (bus *Bus) IteratorQueueBuffer(buf int) {
	if !bus.isInitialized() {
		bus.iteratorQueueBuffer = buf
	}
}

// IteratorResultBuffer may optionally be provided to tweak the buffer size of the results channel for iterator queries.
// This value may have high impact on performance depending on the use case.
// It defaults to 1.
func (bus *Bus) IteratorResultBuffer(buf int) {
	bus.iteratorResultBuffer = buf
}

// InitializeIteratorHandlers initializes the query bus to support iterator queries.
func (bus *Bus) InitializeIteratorHandlers(hdls ...IteratorHandler) {
	if bus.initialize() {
		bus.iteratorHandlers = hdls
		bus.iteratorQueryQueue = make(chan *pendingIteratorQuery, bus.iteratorQueueBuffer)
		for i := 0; i < bus.iteratorWorkerPoolSize; i++ {
			bus.iteratorWorkerUp()
			go bus.iteratorWorker(bus.iteratorQueryQueue, bus.closed)
		}
		atomic.CompareAndSwapUint32(bus.shuttingDown, 1, 0)
	}
}

// Query for a single result or a pre-populated collection.
func (bus *Bus) Query(qry Query) (*Result, error) {
	if err := bus.isValid(qry); err != nil {
		return nil, err
	}

	res, cached := bus.result(qry)
	if cached {
		return res, nil
	}

	return res, bus.query(qry, res)
}

// IteratorQuery uses a channel to iterate the results while they are being populated.
// *Iterator queries are not cached*.
func (bus *Bus) IteratorQuery(qry Query) (*IteratorResult, error) {
	if err := bus.isIteratorValid(qry); err != nil {
		return nil, err
	}

	res := newIteratorResult(bus.iteratorResultBuffer)
	bus.enqueueIteratorQuery(qry, res)
	return res, nil
}

// Shutdown the query bus gracefully.
// *Queries handled while shutting down will be disregarded*.
func (bus *Bus) Shutdown() {
	if atomic.CompareAndSwapUint32(bus.shuttingDown, 0, 1) {
		bus.shutdown()
	}
}

//-----Private Functions------//

func (bus *Bus) initialize() bool {
	return atomic.CompareAndSwapUint32(bus.initialized, 0, 1)
}

func (bus *Bus) isInitialized() bool {
	return atomic.LoadUint32(bus.initialized) == 1
}

func (bus *Bus) isShuttingDown() bool {
	return atomic.LoadUint32(bus.shuttingDown) == 1
}

func (bus *Bus) iteratorWorker(qryQ <-chan *pendingIteratorQuery, closed chan<- bool) {
	for penQry := range qryQ {
		// nil queries are used as signals to break out
		if penQry == nil {
			break
		}

		// wait for a listener
		if penQry.res.waitListener(iteratorListenerTimeout) {
			bus.iteratorQuery(penQry.qry, penQry.res)
			penQry.res.close()
			continue
		}

		bus.error(penQry.qry, fmt.Errorf("the query %T timed out due to lack of result listeners. "+
			"This may happen if a query was issued but the \"Iterate\" function of the result was not handled", penQry.qry))
	}
	closed <- true
}

func (bus *Bus) iteratorQuery(qry Query, res *IteratorResult) {
	for _, hdl := range bus.iteratorHandlers {
		if err := hdl.Handle(qry, res); err != nil {
			bus.error(qry, err)
			return
		}
		if res.propagationStopped() {
			return
		}
	}
	if !res.isHandled() {
		bus.error(qry, fmt.Errorf("no handlers were found for the query %T", qry))
	}
}

func (bus *Bus) enqueueIteratorQuery(qry Query, res *IteratorResult) {
	bus.iteratorQueryQueue <- &pendingIteratorQuery{
		qry: qry,
		res: res,
	}
}

func (bus *Bus) query(qry Query, res *Result) error {
	for _, hdl := range bus.handlers {
		if err := hdl.Handle(qry, res); err != nil {
			bus.error(qry, err)
			return err
		}
		if res.propagationStopped() {
			break
		}
	}

	if !res.isHandled() {
		err := fmt.Errorf("no handlers were found for the query %T", qry)
		bus.error(qry, err)
		return err
	}

	bus.handleCache(qry, res)
	return nil
}

func (bus *Bus) result(qry Query) (*Result, bool) {
	if qry, implements := qry.(Cacheable); implements {
		for _, adp := range bus.cacheAdapters {
			if res := adp.Get(qry); res != nil {
				atomic.CompareAndSwapUint32(res.cached, 0, 1)
				return res, true
			}
		}
		return newCacheableResult(qry), false
	}
	return newResult(), false
}

func (bus *Bus) handleCache(qry Query, res *Result) {
	if qry, implements := qry.(Cacheable); implements && qry.CacheDuration() > 0 {
		at := time.Now()
		cached := false
		for _, adp := range bus.cacheAdapters {
			cached = cached || adp.Set(qry, res, at)
		}
		if cached {
			res.cache(qry, at)
		}
	}
}

func (bus *Bus) iteratorWorkerUp() {
	atomic.AddUint32(bus.iteratorWorkers, 1)
}

func (bus *Bus) iteratorWorkerDown() {
	atomic.AddUint32(bus.iteratorWorkers, ^uint32(0))
}

func (bus *Bus) shutdown() {
	for atomic.LoadUint32(bus.iteratorWorkers) > 0 {
		bus.iteratorQueryQueue <- nil
		<-bus.closed
		bus.iteratorWorkerDown()
	}
	for _, adp := range bus.cacheAdapters {
		adp.Shutdown()
	}
	atomic.CompareAndSwapUint32(bus.initialized, 1, 0)
}

func (bus *Bus) isValid(qry Query) error {
	var err error
	if qry == nil {
		err = errors.New("invalid query")
		bus.error(qry, err)
		return err
	}
	return nil
}

func (bus *Bus) isIteratorValid(qry Query) error {
	err := bus.isValid(qry)
	if err != nil {
		return err
	}
	if !bus.isInitialized() {
		err = errors.New("the query bus is not initialized")
		bus.error(qry, err)
		return err
	}
	if bus.isShuttingDown() {
		err = errors.New("the query bus is shutting down")
		bus.error(qry, err)
		return err
	}
	return nil
}

func (bus *Bus) error(qry Query, err error) {
	for _, errHdl := range bus.errorHandlers {
		errHdl.Handle(qry, err)
	}
}

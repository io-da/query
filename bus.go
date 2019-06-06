package query

import (
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

type Bus struct {
	workerPoolSize int
	queueBuffer    int
	resultBuffer   int
	initialized    *uint32
	ongoingQueries *uint32
	shuttingDown   *uint32
	workers        *uint32
	handlers       []Handler
	queryQueue     chan *pendingQuery
	closed         chan bool
}

// NewBus instantiates the Bus struct.
// The Initialization of the Bus is performed separately (Initialize function) for dependency injection purposes.
func NewBus() *Bus {
	return &Bus{
		workerPoolSize: runtime.GOMAXPROCS(0),
		queueBuffer:    100,
		resultBuffer:   1,
		initialized:    new(uint32),
		ongoingQueries: new(uint32),
		shuttingDown:   new(uint32),
		workers:        new(uint32),
		closed:         make(chan bool),
	}
}

// WorkerPoolSize may optionally be provided to tweak the worker pool size for query queue.
// It can only be adjusted *before* the bus is initialized.
// It defaults to the value returned by runtime.GOMAXPROCS(0).
func (bus *Bus) WorkerPoolSize(workerPoolSize int) {
	if !bus.isInitialized() {
		bus.workerPoolSize = workerPoolSize
	}
}

// QueueBuffer may optionally be provided to tweak the buffer size of the query queue.
// This value may have high impact on performance depending on the use case.
// It can only be adjusted *before* the bus is initialized.
// It defaults to 100.
func (bus *Bus) QueueBuffer(buf int) {
	if !bus.isInitialized() {
		bus.queueBuffer = buf
	}
}

// ResultBuffer may optionally be provided to tweak the buffer size of the results channel.
// This value may have high impact on performance depending on the use case.
// It defaults to 1.
func (bus *Bus) ResultBuffer(buf int) {
	bus.resultBuffer = buf
}

// Initialize the query bus.
func (bus *Bus) Initialize(hdls ...Handler) {
	if bus.initialize() {
		bus.handlers = hdls
		bus.queryQueue = make(chan *pendingQuery, bus.queueBuffer)
		for i := 0; i < bus.workerPoolSize; i++ {
			bus.workerUp()
			go bus.worker(bus.queryQueue, bus.closed)
		}
		atomic.CompareAndSwapUint32(bus.shuttingDown, 1, 0)
	}
}

// Query using a channel to iterate the results while they are being populated.
func (bus *Bus) QueryIterator(qry Query) <-chan Result {
	if qry == nil {
		return bus.errorResult("invalid query")
	}

	if !bus.isInitialized() {
		return bus.errorResult("the query bus is not initialized")
	}

	bus.queryStarted()
	if bus.isShuttingDown() {
		bus.queryFinished()
		return bus.errorResult("the query bus is shutting down")
	}

	resChan := make(chan Result, bus.resultBuffer)
	bus.queryQueue <- &pendingQuery{
		qry:     qry,
		resChan: resChan,
	}
	return resChan
}

// Query for a single result or a pre-populated collection.
func (bus *Bus) Query(qry Query) (Result, error) {
	if qry == nil {
		return nil, errors.New("invalid query")
	}

	if !bus.isInitialized() {
		return nil, errors.New("the query bus is not initialized")
	}

	bus.queryStarted()
	if bus.isShuttingDown() {
		bus.queryFinished()
		return nil, errors.New("the query bus is shutting down")
	}

	resChan := make(chan Result)
	bus.queryQueue <- &pendingQuery{
		qry:     qry,
		resChan: resChan,
	}
	res := <-resChan
	if err, isErr := res.(error); isErr {
		return nil, err
	}
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

func (bus *Bus) worker(queryQueue <-chan *pendingQuery, closed chan<- bool) {
	for penQry := range queryQueue {
		if penQry == nil {
			break
		}

		bus.query(penQry.qry, penQry.resChan)
		close(penQry.resChan)
		bus.queryFinished()
	}
	closed <- true
}

func (bus *Bus) query(qry Query, resChan chan<- Result) {
	for _, hdl := range bus.handlers {
		if hdl.Handle(qry, resChan) {
			return
		}
	}
	resChan <- errors.New(fmt.Sprintf("no handlers were found for the query %T", qry))
}

func (bus *Bus) workerUp() {
	atomic.AddUint32(bus.workers, 1)
}

func (bus *Bus) workerDown() {
	atomic.AddUint32(bus.workers, ^uint32(0))
}

func (bus *Bus) queryStarted() {
	atomic.AddUint32(bus.ongoingQueries, 1)
}

func (bus *Bus) queryFinished() {
	atomic.AddUint32(bus.ongoingQueries, ^uint32(0))
}

func (bus *Bus) shutdown() {
	for atomic.LoadUint32(bus.ongoingQueries) > 0 {
		time.Sleep(time.Microsecond)
	}
	for atomic.LoadUint32(bus.workers) > 0 {
		bus.queryQueue <- nil
		<-bus.closed
		bus.workerDown()
	}
	atomic.CompareAndSwapUint32(bus.initialized, 1, 0)
}

func (bus *Bus) errorResult(error string) <-chan Result {
	errChan := make(chan Result, 1)
	errChan <- errors.New(error)
	close(errChan)
	return errChan
}

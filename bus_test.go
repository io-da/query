package query

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBus_Initialize(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}
	hdl2 := &testHandler{}
	itrHdl := &testIteratorHandler{}
	itrHdl2 := &testIteratorHandler{}

	bus.Handlers(hdl, hdl2)
	if len(bus.handlers) != 2 {
		t.Error("Unexpected number of handlers.")
	}

	bus.InitializeIteratorHandlers(itrHdl, itrHdl2)
	if len(bus.iteratorHandlers) != 2 {
		t.Error("Unexpected number of handlers.")
	}
}

func TestBus_WorkerPoolSize(t *testing.T) {
	bus := NewBus()
	bus.IteratorWorkerPoolSize(10)
	bus.InitializeIteratorHandlers()
	if *bus.iteratorWorkers != 10 {
		t.Error("Unexpected iteratorWorker pool size.")
	}
}

func TestBus_QueueBuffer(t *testing.T) {
	bus := NewBus()
	bus.IteratorQueueBuffer(1000)
	bus.InitializeIteratorHandlers()
	if cap(bus.iteratorQueryQueue) != 1000 {
		t.Error("Unexpected query queue capacity.")
	}
}

func TestBus_ResultBuffer(t *testing.T) {
	bus := NewBus()
	bus.IteratorResultBuffer(1000)
	bus.InitializeIteratorHandlers()
	if bus.iteratorResultBuffer != 1000 {
		t.Error("Unexpected result buffer.")
	}
}

func TestBus_Query(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}
	hdlWErr := &testHandlerWithErrors{}
	hdlCache := &testCacheHandler{}
	bus.Handlers(hdl, hdlWErr, hdlCache)

	_, err := bus.Query(nil)
	if err == nil || err != InvalidQueryError {
		t.Error("Expected InvalidQueryError error.")
	} else if err.Error() != "query: invalid query" {
		t.Error("Unexpected InvalidQueryError message.")
	}

	res, err := bus.Query(testQueryString("test"))
	if err != nil {
		t.Error(err.Error())
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	if len(res.All()) <= 0 || res.All()[0] != "bar" {
		t.Error("Query returned an unexpected value.")
	}

	chQry := &testCacheQuery{}
	res, err = bus.Query(chQry)
	if err != nil {
		t.Error(err.Error())
	}
	// confirm its a fresh result
	if !res.IsFresh() {
		t.Error("Result was expected to be fresh.")
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	chAdt := NewMemoryCacheAdapter()
	bus.CacheAdapters(chAdt)
	// should return a fresh result again since we just replaced the cache adapter
	res, err = bus.Query(chQry)
	if err != nil {
		t.Error(err.Error())
	}
	// confirm its a fresh result
	if !res.IsFresh() {
		t.Error("Result was expected to be fresh.")
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	// should return the cached result and thus avoid the one second processing time
	res, err = bus.Query(chQry)
	if err != nil {
		t.Error(err.Error())
	}
	// confirm its a cached result
	if res.IsFresh() {
		t.Error("Result was expected to be cached.")
	}
	if string(res.CacheKey()) != string(chQry.CacheKey()) {
		t.Error("Result cache key was expected to equal the query cache key.")
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	time.Sleep(time.Second * 2)
	// should return a fresh result since we are waiting more then 1 second (this query is configured to have 1 second cache)
	res, err = bus.Query(chQry)
	if err != nil {
		t.Error(err.Error())
	}
	// confirm its a fresh result
	if !res.IsFresh() {
		t.Error("Result was expected to be fresh.")
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	chAdt.Expire(chQry)
	// should return a fresh result since we are expiring the cache
	res, err = bus.Query(chQry)
	if err != nil {
		t.Error(err.Error())
	}
	// confirm its a fresh result
	if !res.IsFresh() {
		t.Error("Result was expected to be fresh.")
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}

	res, err = bus.Query(&testCacheQuery2{})
	if err != nil {
		t.Error(err.Error())
	}
	// confirm its a fresh result
	if !res.CachedAt().IsZero() {
		t.Error("Result was expected to not be cached.")
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}

	res, err = bus.Query(&testQueryEmptyResult{})
	if err != nil {
		t.Error(err.Error())
	}
	if res.First() != nil {
		t.Error("Query returned an unexpected value.")
	}

	ok := false
	if _, err = bus.Query(&testQueryUnsupported{}); err != nil {
		if err, ok = err.(ErrorNoQueryHandlersFound);
			ok && err.Error() != fmt.Sprintf("query: no handlers were found for the query %T", &testQueryUnsupported{}) {
			t.Error("Unexpected ErrorNoQueryHandlersFound message.")
		}
	}
	if !ok {
		t.Error("Expected ErrorNoQueryHandlersFound error.")
	}

	if _, err = bus.Query(&testQueryError{}); err == nil {
		t.Error("Query was expected to throw an error.")
	}
}

func TestBus_IteratorQuery(t *testing.T) {
	bus := NewBus()
	itrHdl := &testIteratorHandler{}
	itrHdlWErr := &testIteratorHandlerWithErrors{}

	_, err := bus.IteratorQuery(nil)
	if err == nil || err != InvalidQueryError {
		t.Error("Expected InvalidQueryError error.")
	} else if err.Error() != "query: invalid query" {
		t.Error("Unexpected InvalidQueryError message.")
	}
	_, err = bus.IteratorQuery(&testQueryStruct{})
	if err == nil || err != BusNotInitializedError {
		t.Error("Expected BusNotInitializedError error.")
	} else if err.Error() != "query: the bus is not initialized" {
		t.Error("Unexpected BusNotInitializedError message.")
	}

	errHdl := &storeErrorsHandler{
		errs: make(map[string]error),
	}
	bus.ErrorHandlers(errHdl)
	bus.InitializeIteratorHandlers(itrHdl, itrHdlWErr)
	res, err := bus.IteratorQuery(&testQueryStruct{})
	if err != nil {
		t.Error(err.Error())
	}
	if val := <-res.Iterate(); val != "bar" {
		t.Error("Query returned an unexpected value.")
	}

	res, err = bus.IteratorQuery(testQueryString("test"))
	if err != nil {
		t.Error(err.Error())
	}
	if val := <-res.Iterate(); val != "bar" {
		t.Error("Query returned an unexpected value.")
	}

	res, err = bus.IteratorQuery(testQueryString("test"))
	if err != nil {
		t.Error(err.Error())
	}
	// trigger the timeout initialization
	time.Sleep(time.Millisecond)
	if val := <-res.Iterate(); val != "bar" {
		t.Error("Query returned an unexpected value.")
	}

	qryTimeout := testQueryString("test")
	res, err = bus.IteratorQuery(qryTimeout)
	if err != nil {
		t.Error(err.Error())
	}
	// trigger the timeout reset
	time.Sleep(time.Second * 6)
	ok := false
	if err = errHdl.Error(qryTimeout); err != nil {
		if err, ok = err.(ErrorQueryTimedOut);
			ok && err.Error() != fmt.Sprintf("query: the query %T timed out due to lack of result listeners. This may happen if a query was issued but the \"Iterate\" function of the result was not handled", qryTimeout) {
			t.Error("Unexpected ErrorQueryTimedOut message.")
		}
	}
	if !ok {
		t.Error("Expected ErrorQueryTimedOut error.")
	}

	qryUnsup := &testQueryUnsupported{}
	res, err = bus.IteratorQuery(qryUnsup)
	<-res.Iterate()
	err = errHdl.Error(qryUnsup)
	ok = false
	if _, err = bus.Query(&testQueryUnsupported{}); err != nil {
		if err, ok = err.(ErrorNoQueryHandlersFound);
			ok && err.Error() != fmt.Sprintf("query: no handlers were found for the query %T", &testQueryUnsupported{}) {
			t.Error("Unexpected ErrorNoQueryHandlersFound message.")
		}
	}
	if !ok {
		t.Error("Expected ErrorNoQueryHandlersFound error.")
	}

	qryErr := &testQueryError{}
	res, err = bus.IteratorQuery(qryErr)
	<-res.Iterate()
	if err = errHdl.Error(qryErr); err == nil {
		t.Error("Iterator query was expected to throw an error.")
	}
}

func TestBus_Shutdown(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}
	itrHdl := &testIteratorHandler{}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	bus.Handlers(hdl)
	bus.IteratorWorkerPoolSize(10)
	bus.InitializeIteratorHandlers(itrHdl)
	_, err := bus.Query(&testQueryStruct{})
	if err != nil {
		t.Error(err.Error())
	}

	time.AfterFunc(time.Nanosecond*300, func() {
		// graceful shutdown
		bus.Shutdown()
		wg.Done()
	})

	for i := 0; i < 1000; i++ {
		_, _ = bus.Query(&testQueryStruct{})
		_, _ = bus.IteratorQuery(&testQueryStruct{})
	}
	time.Sleep(time.Nanosecond * 300)
	if !bus.isShuttingDown() {
		t.Error("The bus should be shutting down.")
	}
	_, err = bus.IteratorQuery(&testQueryStruct{})
	if err == nil || err != BusIsShuttingDownError {
		t.Error("Expected BusIsShuttingDownError error.")
	} else if err.Error() != "query: the bus is shutting down" {
		t.Error("Unexpected BusIsShuttingDownError message.")
	}
	wg.Wait()
}

func TestBus_HandlerOrder(t *testing.T) {
	bus := NewBus()
	hdls := make([]Handler, 0, 1000)
	for i := 0; i < 1000; i++ {
		hdls = append(hdls, &testHandlerOrder{position: uint32(i)})
	}
	bus.Handlers(hdls...)

	qry := &testHandlerOrderQuery{position: new(uint32), unordered: new(uint32)}
	_, err := bus.Query(qry)
	if err != nil {
		t.Error(err.Error())
	}

	if qry.IsUnordered() {
		t.Error("The Handler order MUST be respected.")
	}
}

func BenchmarkBus_Query(b *testing.B) {
	bus := NewBus()
	bus.Handlers(&testHandler{})
	for n := 0; n < b.N; n++ {
		_, err := bus.Query(&testQueryStruct{})
		if err != nil {
			b.Error(err.Error())
		}
	}
}

func BenchmarkBus_IteratorQuery(b *testing.B) {
	bus := NewBus()
	bus.InitializeIteratorHandlers(&testIteratorHandler{})
	for n := 0; n < b.N; n++ {
		res, err := bus.IteratorQuery(&testQueryStruct{})
		if err != nil {
			b.Error(err.Error())
		}
		<-res.Iterate()
	}
}

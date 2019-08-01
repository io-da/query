package query

import (
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
	if err == nil {
		t.Error("Query was expected to throw an error.")
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

	res, err = bus.Query(&testCacheQuery{})
	if err != nil {
		t.Error(err.Error())
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	// this should return the cached result and thus avoid the one second process time
	res, err = bus.Query(&testCacheQuery{})
	if err != nil {
		t.Error(err.Error())
	}
	if res.First() != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	time.Sleep(time.Second * 2)
	// this should return the fresh result since the cache should have expired by now (thus an other second)
	res, err = bus.Query(&testCacheQuery{})
	if err != nil {
		t.Error(err.Error())
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

	if _, err = bus.Query(&testQueryUnsupported{}); err == nil {
		t.Error("Querying with an unsupported query should trigger an error.")
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
	if err == nil {
		t.Error("Iterator querying an uninitialized bus should trigger an error.")
	}
	_, err = bus.IteratorQuery(&testQueryStruct{})
	if err == nil {
		t.Error("Iterator querying an uninitialized bus should trigger an error.")
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

	res, err = bus.IteratorQuery(testQueryString("test"))
	if err != nil {
		t.Error(err.Error())
	}
	// trigger the timeout reset
	time.Sleep(time.Millisecond)
	if val := <-res.Iterate(); val != "bar" {
		t.Error("Query returned an unexpected value.")
	}

	qry := &testQueryUnsupported{}
	res, err = bus.IteratorQuery(qry)
	<-res.Iterate()
	if err = errHdl.Error(qry); err == nil {
		t.Error("Iterator querying with an unsupported query should trigger an error.")
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
	wg.Wait()

	if !bus.isShuttingDown() {
		t.Error("The bus should be shutting down.")
	}
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

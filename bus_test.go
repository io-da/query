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

	bus.Initialize(hdl, hdl2)
	if len(bus.handlers) != 2 {
		t.Error("Unexpected number of handlers.")
	}
}

func TestBus_WorkerPoolSize(t *testing.T) {
	bus := NewBus()
	bus.WorkerPoolSize(10)
	bus.Initialize()
	if *bus.workers != 10 {
		t.Error("Unexpected worker pool size.")
	}
}

func TestBus_QueueBuffer(t *testing.T) {
	bus := NewBus()
	bus.QueueBuffer(1000)
	bus.Initialize()
	if cap(bus.queryQueue) != 1000 {
		t.Error("Unexpected query queue capacity.")
	}
}

func TestBus_ResultBuffer(t *testing.T) {
	bus := NewBus()
	bus.ResultBuffer(1000)
	bus.Initialize()
	if bus.resultBuffer != 1000 {
		t.Error("Unexpected result buffer.")
	}
}

func TestBus_Query(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}

	_, err := bus.Query(nil)
	if err == nil {
		t.Error("Querying an uninitialized bus should trigger an error.")
	}
	for val := range bus.QueryIterator(nil) {
		if _, isErr := val.(error); !isErr {
			t.Error("Querying an uninitialized bus should trigger an error.")
		}
	}

	_, err = bus.Query(testQueryString("test"))
	if err == nil {
		t.Error("Querying an uninitialized bus should trigger an error.")
	}
	for val := range bus.QueryIterator(&testQueryStruct{}) {
		if _, isErr := val.(error); !isErr {
			t.Error("Querying an uninitialized bus should trigger an error.")
		}
	}

	bus.Initialize(hdl)
	for val := range bus.QueryIterator(&testQueryStruct{}) {
		if val, isErr := val.(error); isErr {
			t.Error(val.Error())
		}
		if val != "bar" {
			t.Error("Query returned an unexpected value.")
		}
	}
	for val := range bus.QueryIterator(testQueryString("test")) {
		if val, isErr := val.(error); isErr {
			t.Error(val.Error())
		}
		if val != "bar" {
			t.Error("Query returned an unexpected value.")
		}
	}

	val, err := bus.Query(testQueryString("test"))
	if val != "bar" {
		t.Error("Query returned an unexpected value.")
	}
	if err != nil {
		t.Error(err.Error())
	}

	_, err = bus.Query(&testQueryUnsupported{})
	if err == nil {
		t.Error("Querying with an unsupported query should trigger an error.")
	}
}

func TestBus_Shutdown(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	bus.Initialize(hdl)
	_, err := bus.Query(&testQueryStruct{})
	if err != nil {
		t.Error(err.Error())
	}

	time.AfterFunc(time.Nanosecond, func() {
		// graceful shutdown
		bus.Shutdown()
		wg.Done()
	})

	for i := 0; i < 1000; i++ {
		_, _ = bus.Query(&testQueryStruct{})
		_ = bus.QueryIterator(&testQueryStruct{})
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
	bus.Initialize(hdls...)

	qry := &testHandlerOrderQuery{position: new(uint32), unordered: new(uint32)}
	_, err := bus.Query(qry)
	if err != nil {
		t.Error(err.Error())
	}

	timeout := time.AfterFunc(time.Second*10, func() {
		t.Fatal("The queries should have been handled by now.")
	})
	timeout.Stop()
	if qry.IsUnordered() {
		t.Error("The Handler order MUST be respected.")
	}
}

func BenchmarkBus_Query(b *testing.B) {
	bus := NewBus()
	bus.Initialize(&testHandler{})
	for n := 0; n < b.N; n++ {
		_, err := bus.Query(&testQueryStruct{})
		if err != nil {
			b.Error(err.Error())
		}
	}
}

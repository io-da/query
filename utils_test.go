package query

import (
	"sync/atomic"
)

//------Querys------//

type testQueryStruct struct {
}

type testQueryString string

type testQueryUnsupported struct {
}

type testHandlerOrderQuery struct {
	position  *uint32
	unordered *uint32
}

func (qry *testHandlerOrderQuery) HandlerPosition(position uint32) {
	if position != atomic.LoadUint32(qry.position) {
		atomic.StoreUint32(qry.unordered, 1)
	}
	atomic.AddUint32(qry.position, 1)

}
func (qry *testHandlerOrderQuery) IsUnordered() bool {
	return atomic.LoadUint32(qry.unordered) == 1
}

//------Handlers------//

type testHandler struct {
}

func (hdl *testHandler) Handle(qry Query, resChan chan<- Result) bool {
	switch qry.(type) {
	case *testQueryStruct, testQueryString:
		resChan <- "bar"
		return true
	}
	return false
}

type testHandlerOrder struct {
	position uint32
}

func (hdl *testHandlerOrder) Handle(qry Query, resChan chan<- Result) bool {
	if qry, listens := qry.(*testHandlerOrderQuery); listens {
		qry.HandlerPosition(hdl.position)
		resChan <- "bar"
		return true
	}
	return false
}

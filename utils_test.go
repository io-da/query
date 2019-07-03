package query

import (
	"errors"
	"sync/atomic"
)

//------Querys------//

type testQueryStruct struct {
}

func (*testQueryStruct) Id() []byte {
	return []byte("UUID")
}

type testQueryError struct {
}

func (*testQueryError) Id() []byte {
	return []byte("UUID")
}

type testQueryString string

func (testQueryString) Id() []byte {
	return []byte("UUID")
}

type testQueryUnsupported struct {
}

func (*testQueryUnsupported) Id() []byte {
	return []byte("UUID")
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
func (*testHandlerOrderQuery) Id() []byte {
	return []byte("UUID")
}

//------Handlers------//

type testHandler struct {
}

func (hdl *testHandler) Handle(qry Query, res chan<- Result) (bool, error) {
	switch qry.(type) {
	case *testQueryStruct, testQueryString:
		res <- "bar"
		return true, nil
	}
	return false, nil
}

type testHandlerWithErrors struct {
}

func (hdl *testHandlerWithErrors) Handle(qry Query, res chan<- Result) (bool, error) {
	switch qry.(type) {
	case *testQueryError:
		return true, errors.New("query failed")
	}
	return false, nil
}

type testHandlerOrder struct {
	position uint32
}

func (hdl *testHandlerOrder) Handle(qry Query, res chan<- Result) (bool, error) {
	if qry, listens := qry.(*testHandlerOrderQuery); listens {
		qry.HandlerPosition(hdl.position)
		res <- "bar"
		return true, nil
	}
	return false, nil
}

//------Error Handlers------//

type storeErrorsHandler struct {
	errs map[string]error
}

func (hdl *storeErrorsHandler) Handle(qry Query, err error) {
	hdl.errs[string(qry.Id())] = err
}

func (hdl *storeErrorsHandler) Error(qry Query) error {
	if err, hasError := hdl.errs[string(qry.Id())]; hasError {
		return err
	}
	return nil
}

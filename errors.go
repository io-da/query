package query

import "fmt"

const (
	InvalidQueryError           = ErrorInvalidQuery("query: invalid query")
	QueryBusNotInitializedError = ErrorQueryBusNotInitialized("query: the query bus is not initialized")
	QueryBusIsShuttingDownError = ErrorQueryBusIsShuttingDown("query: the query bus is shutting down")
)

type ErrorInvalidQuery string

func (e ErrorInvalidQuery) Error() string {
	return string(e)
}

type ErrorQueryBusNotInitialized string

func (e ErrorQueryBusNotInitialized) Error() string {
	return string(e)
}

type ErrorQueryBusIsShuttingDown string

func (e ErrorQueryBusIsShuttingDown) Error() string {
	return string(e)
}

type ErrorNoQueryHandlersFound struct {
	query Query
}

func (e ErrorNoQueryHandlersFound) Error() string {
	return fmt.Sprintf("query: no handlers were found for the query %T", e.query)
}

func NewErrorNoQueryHandlersFound(query Query) ErrorNoQueryHandlersFound {
	return ErrorNoQueryHandlersFound{query: query}
}

type ErrorQueryTimedOut struct {
	query Query
}

func (e ErrorQueryTimedOut) Error() string {
	return fmt.Sprintf("query: the query %T timed out due to lack of result listeners. This may happen if a query was issued but the \"Iterate\" function of the result was not handled", e.query)
}

func NewErrorQueryTimedOut(query Query) ErrorQueryTimedOut {
	return ErrorQueryTimedOut{query: query}
}

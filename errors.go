package query

import "fmt"

// ErrorInvalidQuery is used when invalid queries are handled.
type ErrorInvalidQuery string

// Error returns the string message of ErrorInvalidQuery.
func (e ErrorInvalidQuery) Error() string {
	return string(e)
}

// ErrorBusNotInitialized is used when queries are handled but the bus is not initialized.
type ErrorBusNotInitialized string

// Error returns the string message of ErrorBusNotInitialized.
func (e ErrorBusNotInitialized) Error() string {
	return string(e)
}

// ErrorBusIsShuttingDown is used when queries are handled but the bus is shutting down.
type ErrorBusIsShuttingDown string

// Error returns the string message of ErrorBusIsShuttingDown.
func (e ErrorBusIsShuttingDown) Error() string {
	return string(e)
}

// ErrorNoQueryHandlersFound is used when not a single handler is found for a specific query.
type ErrorNoQueryHandlersFound struct {
	query Query
}

// Error returns the string message of ErrorNoQueryHandlersFound.
func (e ErrorNoQueryHandlersFound) Error() string {
	return fmt.Sprintf("query: no handlers were found for the query %T", e.query)
}

// NewErrorNoQueryHandlersFound creates a new ErrorNoQueryHandlersFound.
func NewErrorNoQueryHandlersFound(query Query) ErrorNoQueryHandlersFound {
	return ErrorNoQueryHandlersFound{query: query}
}

// ErrorQueryTimedOut is used when the handling of a query times out.
type ErrorQueryTimedOut struct {
	query Query
}

// Error returns the string message of ErrorQueryTimedOut.
func (e ErrorQueryTimedOut) Error() string {
	return fmt.Sprintf("query: the query %T timed out due to lack of result listeners. This may happen if a query was issued but the \"Iterate\" function of the result was not handled", e.query)
}

// NewErrorQueryTimedOut creates a new ErrorQueryTimedOut.
func NewErrorQueryTimedOut(query Query) ErrorQueryTimedOut {
	return ErrorQueryTimedOut{query: query}
}

const (
	// InvalidQueryError is a constant equivalent of the ErrorInvalidQuery error.
	InvalidQueryError = ErrorInvalidQuery("query: invalid query")
	// BusNotInitializedError is a constant equivalent of the ErrorBusNotInitialized error.
	BusNotInitializedError = ErrorBusNotInitialized("query: the bus is not initialized")
	// BusIsShuttingDownError is a constant equivalent of the ErrorBusIsShuttingDown error.
	BusIsShuttingDownError = ErrorBusIsShuttingDown("query: the bus is shutting down")
)

package query

// ErrorHandler must be implemented for a type to qualify as an error handler.
type ErrorHandler interface {
	Handle(qry Query, err error)
}

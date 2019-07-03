package query

// Handler must be implemented for a type to qualify as a query handler.
type ErrorHandler interface {
	Handle(qry Query, err error)
}

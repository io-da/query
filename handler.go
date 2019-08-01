package query

// Handler must be implemented for a type to qualify as a query handler.
type Handler interface {
	Handle(qry Query, res *Result) error
}

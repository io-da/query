package query

// IteratorHandler must be implemented for a type to qualify as an iterator query handler.
type IteratorHandler interface {
	Handle(qry Query, res *IteratorResult) error
}

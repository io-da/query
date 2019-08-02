package query

// Query is the interface that must be implemented by any type to be considered a query.
type Query interface {
	ID() []byte
}

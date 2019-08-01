package query

// Query is the interface that must be implemented by all queries
type Query interface {
	ID() []byte
}

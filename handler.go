package query

type Handler interface {
	Handle(qry Query, resChan chan<- Result) bool
}

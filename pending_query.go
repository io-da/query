package query

type pendingQuery struct {
	qry     Query
	resChan chan<- Result
}

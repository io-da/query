package query

type pendingQuery struct {
	qry Query
	res chan<- Result
}

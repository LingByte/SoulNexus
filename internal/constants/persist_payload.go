package constants

// PersistQueryResult is filled synchronously by listeners Connect handlers.
type PersistQueryResult struct {
	Value  any
	Value2 any
	Err    error
}

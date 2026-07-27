package activity

import "context"

// transactor is the minimum interface the use case needs to wrap its
// work in a database transaction. Implemented by postgres.Transactor.
type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

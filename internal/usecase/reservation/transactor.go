package reservation

import "context"

// Transactor is the minimum interface the use case needs to wrap its
// work in a database transaction. Implemented by
// postgres.Transactor. The use case does not depend on database/sql
// directly: this indirection keeps the use case testable with a
// fake Transactor.
type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

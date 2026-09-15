package pass

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByPaidPassesInput is the paginated request for listing
// passes filtered by their is_paid flag.
type ListByPaidPassesInput struct {
	isPaid bool
	limit  int
	offset int
}

func (in ListByPaidPassesInput) IsPaid() bool { return in.isPaid }
func (in ListByPaidPassesInput) Limit() int   { return in.limit }
func (in ListByPaidPassesInput) Offset() int  { return in.offset }

// NewListByPaidPassesInput normalizes pagination. The is_paid flag
// is a boolean and needs no validation.
func NewListByPaidPassesInput(isPaid bool, limit, offset int) (ListByPaidPassesInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListByPaidPassesInput{isPaid: isPaid, limit: limit, offset: offset}, nil
}

// MustNewListByPaidPassesInput panics on error. For tests.
func MustNewListByPaidPassesInput(isPaid bool, limit, offset int) ListByPaidPassesInput {
	in, err := NewListByPaidPassesInput(isPaid, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListByPaidPassesOutput carries the result page, most recent first.
type ListByPaidPassesOutput struct {
	Passes []*domain.Pass
}

// ListByPaidPassesUseCase returns a paginated list of passes with
// the given is_paid flag. Backs the Pagados/Pendientes filter tabs
// on the admin Bonos page.
type ListByPaidPassesUseCase struct {
	repo domain.PassRepository
}

func NewListByPaidPassesUseCase(repo domain.PassRepository) *ListByPaidPassesUseCase {
	return &ListByPaidPassesUseCase{repo: repo}
}

func (uc *ListByPaidPassesUseCase) Execute(ctx context.Context, input ListByPaidPassesInput) (ListByPaidPassesOutput, error) {
	passes, err := uc.repo.ListByPaid(ctx, input.IsPaid(), input.Limit(), input.Offset())
	if err != nil {
		return ListByPaidPassesOutput{}, fmt.Errorf("list passes by paid=%v: %w", input.IsPaid(), err)
	}
	return ListByPaidPassesOutput{Passes: passes}, nil
}

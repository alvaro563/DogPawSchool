package pass

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListAllPassesInput is the paginated request for listing every
// pass in the system. The pagination is already normalized by
// the factory.
type ListAllPassesInput struct {
	limit  int
	offset int
}

func (in ListAllPassesInput) Limit() int  { return in.limit }
func (in ListAllPassesInput) Offset() int { return in.offset }

// NewListAllPassesInput normalizes pagination. Error is always
// nil; the factory exists for uniform signature.
func NewListAllPassesInput(limit, offset int) (ListAllPassesInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListAllPassesInput{limit: limit, offset: offset}, nil
}

// MustNewListAllPassesInput panics on error. For tests.
func MustNewListAllPassesInput(limit, offset int) ListAllPassesInput {
	in, err := NewListAllPassesInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListAllPassesOutput carries the result page, most recent first.
type ListAllPassesOutput struct {
	Passes []*domain.Pass
}

// ListAllPassesUseCase returns a paginated list of all passes in
// the system. In production this should be restricted to admin
// users; the handler documents this with a Swagger TODO.
type ListAllPassesUseCase struct {
	repo domain.PassRepository
}

func NewListAllPassesUseCase(repo domain.PassRepository) *ListAllPassesUseCase {
	return &ListAllPassesUseCase{repo: repo}
}

func (uc *ListAllPassesUseCase) Execute(ctx context.Context, input ListAllPassesInput) (ListAllPassesOutput, error) {
	passes, err := uc.repo.ListAll(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListAllPassesOutput{}, fmt.Errorf("list all passes: %w", err)
	}
	return ListAllPassesOutput{Passes: passes}, nil
}

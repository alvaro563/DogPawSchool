package pass

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListAllPassesInput is the paginated request for listing every
// pass in the system.
type ListAllPassesInput struct {
	Limit  int
	Offset int
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
	limit, offset := NormalizePagination(input.Limit, input.Offset)
	passes, err := uc.repo.ListAll(ctx, limit, offset)
	if err != nil {
		return ListAllPassesOutput{}, fmt.Errorf("list all passes: %w", err)
	}
	return ListAllPassesOutput{Passes: passes}, nil
}

package pass

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByUserPassesInput is the paginated request for listing passes
// owned by a specific user.
type ListByUserPassesInput struct {
	UserID int
	Limit  int
	Offset int
}

// ListByUserPassesOutput carries the result page, most recent first.
type ListByUserPassesOutput struct {
	Passes []*domain.Pass
}

// ListByUserPassesUseCase returns a paginated list of passes owned
// by the given user. The use case does not verify the user exists;
// if the user_id does not exist the repository returns an empty
// slice. Validating user existence requires a UserRepository, which
// is out of scope for this iteration.
type ListByUserPassesUseCase struct {
	repo domain.PassRepository
}

func NewListByUserPassesUseCase(repo domain.PassRepository) *ListByUserPassesUseCase {
	return &ListByUserPassesUseCase{repo: repo}
}

func (uc *ListByUserPassesUseCase) Execute(ctx context.Context, input ListByUserPassesInput) (ListByUserPassesOutput, error) {
	if err := input.validate(); err != nil {
		return ListByUserPassesOutput{}, err
	}
	limit, offset := NormalizePagination(input.Limit, input.Offset)
	passes, err := uc.repo.ListByOwner(ctx, input.UserID, limit, offset)
	if err != nil {
		return ListByUserPassesOutput{}, fmt.Errorf("list passes by user %d: %w", input.UserID, err)
	}
	return ListByUserPassesOutput{Passes: passes}, nil
}

func (input ListByUserPassesInput) validate() error {
	if input.UserID <= 0 {
		return &ValidationError{Field: "user_id"}
	}
	return nil
}

package pass

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListByUserPassesInput is the paginated request for listing
// passes owned by a specific user.
type ListByUserPassesInput struct {
	userID int
	limit  int
	offset int
}

func (in ListByUserPassesInput) UserID() int { return in.userID }
func (in ListByUserPassesInput) Limit() int  { return in.limit }
func (in ListByUserPassesInput) Offset() int { return in.offset }

// NewListByUserPassesInput validates user id and normalizes
// pagination. The use case does not verify the user exists; if
// the user_id does not exist the repository returns an empty
// slice.
func NewListByUserPassesInput(userID, limit, offset int) (ListByUserPassesInput, error) {
	if userID <= 0 {
		return ListByUserPassesInput{}, &ValidationError{Field: "user_id"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByUserPassesInput{userID: userID, limit: limit, offset: offset}, nil
}

// MustNewListByUserPassesInput panics on error. For tests.
func MustNewListByUserPassesInput(userID, limit, offset int) ListByUserPassesInput {
	in, err := NewListByUserPassesInput(userID, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListByUserPassesOutput carries the result page, most recent first.
type ListByUserPassesOutput struct {
	Passes []*domain.Pass
}

// ListByUserPassesUseCase returns a paginated list of passes owned
// by the given user.
type ListByUserPassesUseCase struct {
	repo domain.PassRepository
}

func NewListByUserPassesUseCase(repo domain.PassRepository) *ListByUserPassesUseCase {
	return &ListByUserPassesUseCase{repo: repo}
}

func (uc *ListByUserPassesUseCase) Execute(ctx context.Context, input ListByUserPassesInput) (ListByUserPassesOutput, error) {
	passes, err := uc.repo.ListByOwner(ctx, input.UserID(), input.Limit(), input.Offset())
	if err != nil {
		return ListByUserPassesOutput{}, fmt.Errorf("list passes by user %d: %w", input.UserID(), err)
	}
	return ListByUserPassesOutput{Passes: passes}, nil
}

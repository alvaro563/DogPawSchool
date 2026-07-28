package user

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUsersInput is the validated paginated request for listing every
// user (admin view). The pagination is already normalized by the
// factory.
type ListUsersInput struct {
	limit  int
	offset int
}

func (in ListUsersInput) Limit() int  { return in.limit }
func (in ListUsersInput) Offset() int { return in.offset }

// NewListUsersInput normalizes pagination. The error is always nil for
// pure-pagination inputs; it is returned to keep the factory signature
// uniform across the package.
func NewListUsersInput(limit, offset int) (ListUsersInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListUsersInput{limit: limit, offset: offset}, nil
}

// MustNewListUsersInput panics on error. For tests.
func MustNewListUsersInput(limit, offset int) ListUsersInput {
	in, err := NewListUsersInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// ListUsersOutput carries the page of users.
type ListUsersOutput struct {
	Users []*domain.User
}

// ListUsersUseCase loads a paginated slice of users for the admin
// view. No filters are applied at the use case layer; the caller is
// expected to do further client-side filtering if needed.
type ListUsersUseCase struct {
	repo domain.UserRepository
}

func NewListUsersUseCase(repo domain.UserRepository) *ListUsersUseCase {
	return &ListUsersUseCase{repo: repo}
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, input ListUsersInput) (ListUsersOutput, error) {
	users, err := uc.repo.ListAllPaged(ctx, input.Limit(), input.Offset())
	if err != nil {
		return ListUsersOutput{}, fmt.Errorf("list users: %w", err)
	}
	return ListUsersOutput{Users: users}, nil
}

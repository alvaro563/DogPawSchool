package user

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListUserEmailsOutput carries every registered email address.
type ListUserEmailsOutput struct {
	Emails []string
}

// ListUserEmailsUseCase returns the email of every user in the
// system, ordered by user id ascending. There is nothing to validate
// (no input), so Execute takes only the context; this keeps the type
// honest about having zero parameters. The caller is responsible for
// authorizing the request (admin-only endpoint).
type ListUserEmailsUseCase struct {
	repo domain.UserRepository
}

func NewListUserEmailsUseCase(repo domain.UserRepository) *ListUserEmailsUseCase {
	return &ListUserEmailsUseCase{repo: repo}
}

func (uc *ListUserEmailsUseCase) Execute(ctx context.Context) (ListUserEmailsOutput, error) {
	emails, err := uc.repo.ListAllEmails(ctx)
	if err != nil {
		return ListUserEmailsOutput{}, fmt.Errorf("list user emails: %w", err)
	}
	return ListUserEmailsOutput{Emails: emails}, nil
}

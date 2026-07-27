package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// DeleteDogInput is the validated command to delete a dog. Only
// NewDeleteDogInput can construct one.
type DeleteDogInput struct {
	id int
}

func (in DeleteDogInput) ID() int { return in.id }

// NewDeleteDogInput validates id > 0.
func NewDeleteDogInput(id int) (DeleteDogInput, error) {
	if id <= 0 {
		return DeleteDogInput{}, &ValidationError{Field: "id"}
	}
	return DeleteDogInput{id: id}, nil
}

// MustNewDeleteDogInput panics on validation error. For tests.
func MustNewDeleteDogInput(id int) DeleteDogInput {
	in, err := NewDeleteDogInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

// DeleteDogOutput is empty: a successful delete returns no payload.
type DeleteDogOutput struct{}

// DeleteDogUseCase removes a dog aggregate by id. Cascades to the
// associated dog_incompatibilities and reservations rows are handled
// at the DB level by ON DELETE CASCADE foreign keys.
type DeleteDogUseCase struct {
	repo domain.DogRepository
}

func NewDeleteDogUseCase(repo domain.DogRepository) *DeleteDogUseCase {
	return &DeleteDogUseCase{repo: repo}
}

func (uc *DeleteDogUseCase) Execute(ctx context.Context, input DeleteDogInput) (DeleteDogOutput, error) {
	if err := uc.repo.Delete(ctx, input.ID()); err != nil {
		return DeleteDogOutput{}, fmt.Errorf("delete dog: %w", err)
	}
	return DeleteDogOutput{}, nil
}

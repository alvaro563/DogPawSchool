package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// DeleteDogInput is the input for DeleteDogUseCase.
type DeleteDogInput struct {
	ID int
}

// DeleteDogOutput is empty: a successful delete returns no payload.
type DeleteDogOutput struct{}

// DeleteDogUseCase removes a dog aggregate by id. Cascades to the
// associated dog_incompatibilities and reservations rows are handled at the
// DB level by ON DELETE CASCADE foreign keys, so the use case does not need
// to touch the dog_incompatibilities table directly.
type DeleteDogUseCase struct {
	repo domain.DogRepository
}

func NewDeleteDogUseCase(repo domain.DogRepository) *DeleteDogUseCase {
	return &DeleteDogUseCase{repo: repo}
}

func (uc *DeleteDogUseCase) Execute(ctx context.Context, in DeleteDogInput) (DeleteDogOutput, error) {
	if in.ID <= 0 {
		return DeleteDogOutput{}, &ValidationError{Field: "id"}
	}
	if err := uc.repo.Delete(ctx, in.ID); err != nil {
		return DeleteDogOutput{}, fmt.Errorf("delete dog: %w", err)
	}
	return DeleteDogOutput{}, nil
}

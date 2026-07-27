package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

type GetDogInput struct {
	id int
}

func (in GetDogInput) ID() int { return in.id }

func NewGetDogInput(id int) (GetDogInput, error) {
	if id <= 0 {
		return GetDogInput{}, &ValidationError{Field: "id"}
	}
	return GetDogInput{id: id}, nil
}

func MustNewGetDogInput(id int) GetDogInput {
	in, err := NewGetDogInput(id)
	if err != nil {
		panic(err)
	}
	return in
}

type GetDogOutput struct {
	Dog *domain.Dog
}

type GetDogUseCase struct {
	repo domain.DogRepository
}

func NewGetDogUseCase(repo domain.DogRepository) *GetDogUseCase {
	return &GetDogUseCase{repo: repo}
}

func (uc *GetDogUseCase) Execute(ctx context.Context, input GetDogInput) (GetDogOutput, error) {
	dog, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return GetDogOutput{}, fmt.Errorf("get dog %d: %w", input.ID(), err)
	}
	if dog == nil {
		return GetDogOutput{}, ErrNotFound
	}
	return GetDogOutput{Dog: dog}, nil
}

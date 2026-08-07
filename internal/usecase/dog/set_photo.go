package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

type SetDogPhotoInput struct {
	id       int
	photoURL string
}

func (in SetDogPhotoInput) ID() int       { return in.id }
func (in SetDogPhotoInput) PhotoURL() string { return in.photoURL }

func NewSetDogPhotoInput(id int, photoURL string) (SetDogPhotoInput, error) {
	if id <= 0 {
		return SetDogPhotoInput{}, &ValidationError{Field: "id"}
	}
	return SetDogPhotoInput{id: id, photoURL: photoURL}, nil
}

func MustNewSetDogPhotoInput(id int, photoURL string) SetDogPhotoInput {
	in, err := NewSetDogPhotoInput(id, photoURL)
	if err != nil {
		panic(err)
	}
	return in
}

type SetDogPhotoOutput struct {
	ID       int
	PhotoURL string
}

type SetDogPhotoUseCase struct {
	transactor Transactor
	repo       domain.DogRepository
}

func NewSetDogPhotoUseCase(transactor Transactor, repo domain.DogRepository) *SetDogPhotoUseCase {
	return &SetDogPhotoUseCase{transactor: transactor, repo: repo}
}

func (uc *SetDogPhotoUseCase) Execute(ctx context.Context, input SetDogPhotoInput) (SetDogPhotoOutput, error) {
	var out SetDogPhotoOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.repo.GetByIDForUpdate(txCtx, input.ID())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("set dog photo: %w", err)
		}

		dog.SetPhotoURL(input.PhotoURL())
		if err := uc.repo.Update(txCtx, dog); err != nil {
			return fmt.Errorf("set dog photo: %w", err)
		}
		out = SetDogPhotoOutput{
			ID:       dog.ID(),
			PhotoURL: dog.PhotoURL(),
		}
		return nil
	})
	return out, err
}

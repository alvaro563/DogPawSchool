package dog

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// ModifyDogInput is the validated command to apply a partial update
// to an existing dog. Fields are private: only NewModifyDogInput
// can construct one. The patch *values* are validated by the domain
// (domain.Dog.ApplyPatch) — defense in depth.
type ModifyDogInput struct {
	id    int
	patch domain.DogPatch
}

func (in ModifyDogInput) ID() int                { return in.id }
func (in ModifyDogInput) Patch() domain.DogPatch { return in.patch }

// NewModifyDogInput validates id > 0 and stores the patch. An empty
// patch is allowed (Execute short-circuits to a no-op, preserving
// the current behavior).
func NewModifyDogInput(id int, patch domain.DogPatch) (ModifyDogInput, error) {
	if id <= 0 {
		return ModifyDogInput{}, &ValidationError{Field: "id"}
	}
	return ModifyDogInput{id: id, patch: patch}, nil
}

// MustNewModifyDogInput panics on validation error. For tests.
func MustNewModifyDogInput(id int, patch domain.DogPatch) ModifyDogInput {
	in, err := NewModifyDogInput(id, patch)
	if err != nil {
		panic(err)
	}
	return in
}

// ModifyDogOutput carries the post-mutation dog id.
type ModifyDogOutput struct {
	ID int
}

// ModifyDogUseCase applies a partial update to a dog. An empty patch
// is a no-op and returns the unmodified dog without touching the DB.
// Mutations run inside a single transaction with FOR UPDATE on the
// dog row, preventing lost updates (B4).
type ModifyDogUseCase struct {
	transactor Transactor
	repo       domain.DogRepository
}

func NewModifyDogUseCase(transactor Transactor, repo domain.DogRepository) *ModifyDogUseCase {
	return &ModifyDogUseCase{transactor: transactor, repo: repo}
}

func (uc *ModifyDogUseCase) Execute(ctx context.Context, input ModifyDogInput) (ModifyDogOutput, error) {
	patch := input.Patch()
	if isEmptyPatch(patch) {
		dog, err := uc.repo.GetByID(ctx, input.ID())
		if err != nil {
			return ModifyDogOutput{}, fmt.Errorf("get dog %d: %w", input.ID(), err)
		}
		if dog == nil {
			return ModifyDogOutput{}, ErrNotFound
		}
		return ModifyDogOutput{ID: dog.ID()}, nil
	}

	var out ModifyDogOutput
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		dog, err := uc.repo.GetByIDForUpdate(txCtx, input.ID())
		if err != nil {
			return fmt.Errorf("get dog %d: %w", input.ID(), err)
		}
		if dog == nil {
			return ErrNotFound
		}

		if err := dog.ApplyPatch(patch); err != nil {
			var validationErr *domain.DogValidationError
			if errors.As(err, &validationErr) {
				return &ValidationError{Field: validationErr.Field}
			}
			return err
		}

		if err := uc.repo.Update(txCtx, dog); err != nil {
			return fmt.Errorf("update dog %d: %w", input.ID(), err)
		}
		out = ModifyDogOutput{ID: dog.ID()}
		return nil
	})
	return out, err
}

func isEmptyPatch(patch domain.DogPatch) bool {
	return patch.Name == nil && patch.Breed == nil && patch.AgeInMonths == nil &&
		patch.Sex == nil && patch.Passport == nil && patch.WeightKg == nil &&
		patch.Neutered == nil && patch.Heat == nil && patch.PhotoURL == nil &&
		patch.MedicalNotes == nil && patch.EducatorNotes == nil && patch.IsActive == nil
}

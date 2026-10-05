package activity

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// maxBatchSessions caps how many sessions a single recurring create
// may materialize (52 weekly occurrences ≈ one year).
const maxBatchSessions = 52

// BatchRegisterActivityInput is the validated command to create a
// recurring activity: one row per entry in dates, all inside a single
// transaction. The shared activity fields are carried by the embedded
// RegisterActivityInput (built by its own validating factory); the
// per-session timestamps come exclusively from dates — in batch mode
// the top-level date field is ignored (the client still sends it as
// the first occurrence, but every row uses its own entry).
type BatchRegisterActivityInput struct {
	register RegisterActivityInput
	dates    []time.Time
}

func (in BatchRegisterActivityInput) Register() RegisterActivityInput { return in.register }
func (in BatchRegisterActivityInput) Dates() []time.Time              { return in.dates }

// NewBatchRegisterActivityInput validates the dates vector:
// at least one, at most maxBatchSessions, no zero timestamps and
// strictly increasing (which also rejects duplicates). It reuses an
// already-validated RegisterActivityInput for the shared fields.
func NewBatchRegisterActivityInput(register RegisterActivityInput, dates []time.Time) (BatchRegisterActivityInput, error) {
	if len(dates) == 0 {
		return BatchRegisterActivityInput{}, &ValidationError{Field: "dates"}
	}
	if len(dates) > maxBatchSessions {
		return BatchRegisterActivityInput{}, &ValidationError{Field: "dates"}
	}
	for i, date := range dates {
		if date.IsZero() {
			return BatchRegisterActivityInput{}, &ValidationError{Field: fmt.Sprintf("dates[%d]", i)}
		}
		if i > 0 && !date.After(dates[i-1]) {
			return BatchRegisterActivityInput{}, &ValidationError{Field: "dates"}
		}
	}
	return BatchRegisterActivityInput{register: register, dates: dates}, nil
}

// MustNewBatchRegisterActivityInput panics on validation error. For tests.
func MustNewBatchRegisterActivityInput(register RegisterActivityInput, dates []time.Time) BatchRegisterActivityInput {
	in, err := NewBatchRegisterActivityInput(register, dates)
	if err != nil {
		panic(err)
	}
	return in
}

// BatchRegisterActivityOutput reports every id created, in the same
// order as the submitted dates.
type BatchRegisterActivityOutput struct {
	IDs []int
}

// BatchRegisterActivityUseCase materializes a recurring activity as
// N independent activity rows, atomically: either every session is
// created or none is. Each row is a first-class activity, so it can
// afterwards be edited, closed or deleted on its own.
type BatchRegisterActivityUseCase struct {
	transactor transactor
	repo       domain.ActivityRepository
	dogRepo    domain.DogRepository
}

func NewBatchRegisterActivityUseCase(
	transactor transactor,
	repo domain.ActivityRepository,
	dogRepo domain.DogRepository,
) *BatchRegisterActivityUseCase {
	return &BatchRegisterActivityUseCase{transactor: transactor, repo: repo, dogRepo: dogRepo}
}

func (uc *BatchRegisterActivityUseCase) Execute(ctx context.Context, input BatchRegisterActivityInput) (BatchRegisterActivityOutput, error) {
	// The target dog is validated once, outside the transaction: a
	// bad dog_id must not even open a tx.
	if err := verifyTargetDog(ctx, uc.dogRepo, input.register.DogID()); err != nil {
		return BatchRegisterActivityOutput{}, err
	}

	var ids []int
	err := uc.transactor.WithinTx(ctx, func(txCtx context.Context) error {
		ids = make([]int, 0, len(input.dates))
		for i, date := range input.dates {
			activity, err := domain.NewActivity(
				0, input.register.Name(), input.register.Description(), input.register.Location(),
				input.register.ActivityType(), input.register.MaxCapacity(), input.register.DurationInHours(),
				date, input.register.DogID(), input.register.SizeTarget())
			if err != nil {
				return err
			}
			id, err := uc.repo.Create(txCtx, activity)
			if err != nil {
				return fmt.Errorf("register session %d: %w", i, err)
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return BatchRegisterActivityOutput{}, err
	}
	return BatchRegisterActivityOutput{IDs: ids}, nil
}

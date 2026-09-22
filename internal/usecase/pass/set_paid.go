package pass

import (
	"context"
	"errors"
	"fmt"

	"dogpaw/internal/domain"
)

// SetPassPaidInput is the validated command to update the
// is_paid flag of an existing pass. Admin-only; the caller is
// responsible for enforcing authorization at the handler layer.
type SetPassPaidInput struct {
	id     int
	isPaid bool
}

func (in SetPassPaidInput) ID() int      { return in.id }
func (in SetPassPaidInput) IsPaid() bool { return in.isPaid }

// NewSetPassPaidInput validates id > 0. The is_paid flag itself is
// a boolean and needs no validation: the only two legal values are
// true and false, both meaningful (paid and unpaid).
func NewSetPassPaidInput(id int, isPaid bool) (SetPassPaidInput, error) {
	if id <= 0 {
		return SetPassPaidInput{}, &ValidationError{Field: "id"}
	}
	return SetPassPaidInput{id: id, isPaid: isPaid}, nil
}

// MustNewSetPassPaidInput panics on validation error. For tests.
func MustNewSetPassPaidInput(id int, isPaid bool) SetPassPaidInput {
	in, err := NewSetPassPaidInput(id, isPaid)
	if err != nil {
		panic(err)
	}
	return in
}

// SetPassPaidOutput carries the post-mutation pass. The full domain
// object is returned so the handler can serialize it directly.
type SetPassPaidOutput struct {
	Pass *domain.Pass
}

// SetPassPaidUseCase flips (or sets) the is_paid flag of a pass.
// Implemented as a Modify+Update flow rather than a dedicated SQL
// UPDATE so that the domain-level audit story stays consistent with
// every other mutation: the aggregate is loaded, mutated through
// ApplyPatch, and persisted with PassRepository.Update. The DB
// trigger that bumps updated_at fires on either path.
type SetPassPaidUseCase struct {
	repo domain.PassRepository
}

func NewSetPassPaidUseCase(repo domain.PassRepository) *SetPassPaidUseCase {
	return &SetPassPaidUseCase{repo: repo}
}

func (uc *SetPassPaidUseCase) Execute(ctx context.Context, input SetPassPaidInput) (SetPassPaidOutput, error) {
	pass, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return SetPassPaidOutput{}, fmt.Errorf("get pass %d: %w", input.ID(), err)
	}
	if pass == nil {
		return SetPassPaidOutput{}, ErrNotFound
	}

	isPaid := input.IsPaid()
	if err := pass.ApplyPatch(domain.PassPatch{IsPaid: &isPaid}); err != nil {
		return SetPassPaidOutput{}, err
	}

	// Snapshot the pass updated_at for the SQL guard. ApplyPatch
	// does not touch updatedAt, so this value is unchanged from
	// the initial GetByID.
	passUpdatedAt := pass.UpdatedAt()

	if err := uc.repo.Update(ctx, pass, passUpdatedAt); err != nil {
		if errors.Is(err, domain.ErrPassStateChanged) {
			return SetPassPaidOutput{}, fmt.Errorf("update pass %d: %w", input.ID(), err)
		}
		return SetPassPaidOutput{}, fmt.Errorf("update pass %d: %w", input.ID(), err)
	}

	// Re-fetch to surface the post-update updatedAt (set by the DB
	// trigger on every UPDATE). Without this, the response would
	// carry the pre-update updatedAt from the in-memory pass.
	updatedPass, err := uc.repo.GetByID(ctx, input.ID())
	if err != nil {
		return SetPassPaidOutput{}, fmt.Errorf("get updated pass %d: %w", input.ID(), err)
	}
	return SetPassPaidOutput{Pass: updatedPass}, nil
}

package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// DogWithOwner is the read-model projection that pairs a Dog with
// its owner's display name. It is defined here (not in domain) per
// the architectural rule that this read shape lives with the use
// case that produces it.
type DogWithOwner struct {
	Dog       *domain.Dog
	OwnerName string
}

// DogWithOwnerLister is the narrow port the use case depends on.
// It deliberately does not extend domain.DogRepository so existing
// mocks of that interface (e.g. mockDogRepository in helpers_test.go)
// remain valid: this new use case is wired in cmd/api through a small
// adapter that converts the infra's *DogWithOwnerRow into DogWithOwner.
type DogWithOwnerLister interface {
	ListActiveWithOwner(ctx context.Context, limit, offset int) ([]*DogWithOwner, error)
}

// ListActiveDogsWithOwnerInput is the validated paginated request for
// the enriched active-dogs listing. Pagination semantics mirror
// ListActiveDogsInput so the handler can reuse the same parsing path.
type ListActiveDogsWithOwnerInput struct {
	limit  int
	offset int
}

func (in ListActiveDogsWithOwnerInput) Limit() int  { return in.limit }
func (in ListActiveDogsWithOwnerInput) Offset() int { return in.offset }

// NewListActiveDogsWithOwnerInput accepts raw limit/offset values and
// returns the validated input. Pagination bounds are applied by the
// repo, not here, mirroring the existing list use cases.
func NewListActiveDogsWithOwnerInput(limit, offset int) (ListActiveDogsWithOwnerInput, error) {
	return ListActiveDogsWithOwnerInput{limit: limit, offset: offset}, nil
}

// ListActiveDogsWithOwnerOutput is the enriched list result.
type ListActiveDogsWithOwnerOutput struct {
	Items []*DogWithOwner
}

// ListActiveDogsWithOwnerUseCase exposes the active-dogs-with-owner
// read. It depends on the narrow DogWithOwnerLister port so adding
// it does not break any existing test/mock of DogRepository.
type ListActiveDogsWithOwnerUseCase struct {
	repo DogWithOwnerLister
}

func NewListActiveDogsWithOwnerUseCase(repo DogWithOwnerLister) *ListActiveDogsWithOwnerUseCase {
	return &ListActiveDogsWithOwnerUseCase{repo: repo}
}

func (uc *ListActiveDogsWithOwnerUseCase) Execute(ctx context.Context, input ListActiveDogsWithOwnerInput) (ListActiveDogsWithOwnerOutput, error) {
	items, err := uc.repo.ListActiveWithOwner(ctx, input.limit, input.offset)
	if err != nil {
		return ListActiveDogsWithOwnerOutput{}, fmt.Errorf("list active dogs with owner: %w", err)
	}
	return ListActiveDogsWithOwnerOutput{Items: items}, nil
}

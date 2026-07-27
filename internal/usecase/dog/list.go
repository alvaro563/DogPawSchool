package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

// ListAllDogsInput is the validated paginated request for listing
// every dog. The pagination is already normalized by the factory.
type ListAllDogsInput struct {
	limit  int
	offset int
}

func (in ListAllDogsInput) Limit() int  { return in.limit }
func (in ListAllDogsInput) Offset() int { return in.offset }

// NewListAllDogsInput normalizes pagination. The error is always nil
// for pure-pagination inputs; it is returned to keep the factory
// signature uniform across the package.
func NewListAllDogsInput(limit, offset int) (ListAllDogsInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListAllDogsInput{limit: limit, offset: offset}, nil
}

// MustNewListAllDogsInput panics on error. For tests.
func MustNewListAllDogsInput(limit, offset int) ListAllDogsInput {
	in, err := NewListAllDogsInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListAllDogsOutput struct {
	Dogs []*domain.Dog
}

type ListAllDogsUseCase struct {
	repo domain.DogRepository
}

func NewListAllDogsUseCase(repo domain.DogRepository) *ListAllDogsUseCase {
	return &ListAllDogsUseCase{repo: repo}
}

func (uc *ListAllDogsUseCase) Execute(ctx context.Context, input ListAllDogsInput) (ListAllDogsOutput, error) {
	dogs, err := uc.repo.ListAll(ctx, false, input.Limit(), input.Offset())
	if err != nil {
		return ListAllDogsOutput{}, fmt.Errorf("list all dogs: %w", err)
	}
	return ListAllDogsOutput{Dogs: dogs}, nil
}

// ListActiveDogsInput is the validated paginated request for listing
// active dogs.
type ListActiveDogsInput struct {
	limit  int
	offset int
}

func (in ListActiveDogsInput) Limit() int  { return in.limit }
func (in ListActiveDogsInput) Offset() int { return in.offset }

func NewListActiveDogsInput(limit, offset int) (ListActiveDogsInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListActiveDogsInput{limit: limit, offset: offset}, nil
}

func MustNewListActiveDogsInput(limit, offset int) ListActiveDogsInput {
	in, err := NewListActiveDogsInput(limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListActiveDogsOutput struct {
	Dogs []*domain.Dog
}

type ListActiveDogsUseCase struct {
	repo domain.DogRepository
}

func NewListActiveDogsUseCase(repo domain.DogRepository) *ListActiveDogsUseCase {
	return &ListActiveDogsUseCase{repo: repo}
}

func (uc *ListActiveDogsUseCase) Execute(ctx context.Context, input ListActiveDogsInput) (ListActiveDogsOutput, error) {
	dogs, err := uc.repo.ListAll(ctx, true, input.Limit(), input.Offset())
	if err != nil {
		return ListActiveDogsOutput{}, fmt.Errorf("list active dogs: %w", err)
	}
	return ListActiveDogsOutput{Dogs: dogs}, nil
}

// ListByOwnerInput is the validated request for listing dogs owned
// by a given user.
type ListByOwnerInput struct {
	ownerID int
	limit   int
	offset  int
}

func (in ListByOwnerInput) OwnerID() int { return in.ownerID }
func (in ListByOwnerInput) Limit() int   { return in.limit }
func (in ListByOwnerInput) Offset() int  { return in.offset }

func NewListByOwnerInput(ownerID, limit, offset int) (ListByOwnerInput, error) {
	if ownerID <= 0 {
		return ListByOwnerInput{}, &ValidationError{Field: "owner_id"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByOwnerInput{ownerID: ownerID, limit: limit, offset: offset}, nil
}

func MustNewListByOwnerInput(ownerID, limit, offset int) ListByOwnerInput {
	in, err := NewListByOwnerInput(ownerID, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListByOwnerOutput struct {
	Dogs []*domain.Dog
}

type ListByOwnerUseCase struct {
	repo domain.DogRepository
}

func NewListByOwnerUseCase(repo domain.DogRepository) *ListByOwnerUseCase {
	return &ListByOwnerUseCase{repo: repo}
}

func (uc *ListByOwnerUseCase) Execute(ctx context.Context, input ListByOwnerInput) (ListByOwnerOutput, error) {
	dogs, err := uc.repo.ListByOwner(ctx, input.OwnerID(), input.Limit(), input.Offset())
	if err != nil {
		return ListByOwnerOutput{}, fmt.Errorf("list dogs by owner: %w", err)
	}
	return ListByOwnerOutput{Dogs: dogs}, nil
}

// ListByIncompatibilityInput is the validated request for listing
// dogs that have a given incompatibility attached.
type ListByIncompatibilityInput struct {
	incompatibilityID int
	limit             int
	offset            int
}

func (in ListByIncompatibilityInput) IncompatibilityID() int { return in.incompatibilityID }
func (in ListByIncompatibilityInput) Limit() int             { return in.limit }
func (in ListByIncompatibilityInput) Offset() int            { return in.offset }

func NewListByIncompatibilityInput(incompatibilityID, limit, offset int) (ListByIncompatibilityInput, error) {
	if incompatibilityID <= 0 {
		return ListByIncompatibilityInput{}, &ValidationError{Field: "incompatibility_id"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByIncompatibilityInput{incompatibilityID: incompatibilityID, limit: limit, offset: offset}, nil
}

func MustNewListByIncompatibilityInput(incompatibilityID, limit, offset int) ListByIncompatibilityInput {
	in, err := NewListByIncompatibilityInput(incompatibilityID, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListByIncompatibilityOutput struct {
	Dogs []*domain.Dog
}

type ListByIncompatibilityUseCase struct {
	repo domain.DogRepository
}

func NewListByIncompatibilityUseCase(repo domain.DogRepository) *ListByIncompatibilityUseCase {
	return &ListByIncompatibilityUseCase{repo: repo}
}

func (uc *ListByIncompatibilityUseCase) Execute(ctx context.Context, input ListByIncompatibilityInput) (ListByIncompatibilityOutput, error) {
	dogs, err := uc.repo.ListByIncompatibility(ctx, input.IncompatibilityID(), input.Limit(), input.Offset())
	if err != nil {
		return ListByIncompatibilityOutput{}, fmt.Errorf("list by incompatibility: %w", err)
	}
	return ListByIncompatibilityOutput{Dogs: dogs}, nil
}

// ListByBreedInput is the validated request for listing dogs of a
// given breed.
type ListByBreedInput struct {
	breed  string
	limit  int
	offset int
}

func (in ListByBreedInput) Breed() string { return in.breed }
func (in ListByBreedInput) Limit() int    { return in.limit }
func (in ListByBreedInput) Offset() int   { return in.offset }

func NewListByBreedInput(breed string, limit, offset int) (ListByBreedInput, error) {
	if breed == "" {
		return ListByBreedInput{}, &ValidationError{Field: "breed"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByBreedInput{breed: breed, limit: limit, offset: offset}, nil
}

func MustNewListByBreedInput(breed string, limit, offset int) ListByBreedInput {
	in, err := NewListByBreedInput(breed, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListByBreedOutput struct {
	Dogs []*domain.Dog
}

type ListByBreedUseCase struct {
	repo domain.DogRepository
}

func NewListByBreedUseCase(repo domain.DogRepository) *ListByBreedUseCase {
	return &ListByBreedUseCase{repo: repo}
}

func (uc *ListByBreedUseCase) Execute(ctx context.Context, input ListByBreedInput) (ListByBreedOutput, error) {
	dogs, err := uc.repo.ListByBreed(ctx, input.Breed(), input.Limit(), input.Offset())
	if err != nil {
		return ListByBreedOutput{}, fmt.Errorf("list by breed: %w", err)
	}
	return ListByBreedOutput{Dogs: dogs}, nil
}

// ListBySexInput is the validated request for listing dogs of a
// given sex.
type ListBySexInput struct {
	sex    domain.Sex
	limit  int
	offset int
}

func (in ListBySexInput) Sex() domain.Sex { return in.sex }
func (in ListBySexInput) Limit() int      { return in.limit }
func (in ListBySexInput) Offset() int     { return in.offset }

func NewListBySexInput(sex domain.Sex, limit, offset int) (ListBySexInput, error) {
	if !sex.IsValid() {
		return ListBySexInput{}, &ValidationError{Field: "sex"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListBySexInput{sex: sex, limit: limit, offset: offset}, nil
}

func MustNewListBySexInput(sex domain.Sex, limit, offset int) ListBySexInput {
	in, err := NewListBySexInput(sex, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListBySexOutput struct {
	Dogs []*domain.Dog
}

type ListBySexUseCase struct {
	repo domain.DogRepository
}

func NewListBySexUseCase(repo domain.DogRepository) *ListBySexUseCase {
	return &ListBySexUseCase{repo: repo}
}

func (uc *ListBySexUseCase) Execute(ctx context.Context, input ListBySexInput) (ListBySexOutput, error) {
	dogs, err := uc.repo.ListBySex(ctx, input.Sex(), input.Limit(), input.Offset())
	if err != nil {
		return ListBySexOutput{}, fmt.Errorf("list by sex: %w", err)
	}
	return ListBySexOutput{Dogs: dogs}, nil
}

// ListByNeuteredInput is the validated request for listing dogs by
// the neutered flag.
type ListByNeuteredInput struct {
	neutered bool
	limit    int
	offset   int
}

func (in ListByNeuteredInput) Neutered() bool { return in.neutered }
func (in ListByNeuteredInput) Limit() int     { return in.limit }
func (in ListByNeuteredInput) Offset() int    { return in.offset }

func NewListByNeuteredInput(neutered bool, limit, offset int) (ListByNeuteredInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListByNeuteredInput{neutered: neutered, limit: limit, offset: offset}, nil
}

func MustNewListByNeuteredInput(neutered bool, limit, offset int) ListByNeuteredInput {
	in, err := NewListByNeuteredInput(neutered, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListByNeuteredOutput struct {
	Dogs []*domain.Dog
}

type ListByNeuteredUseCase struct {
	repo domain.DogRepository
}

func NewListByNeuteredUseCase(repo domain.DogRepository) *ListByNeuteredUseCase {
	return &ListByNeuteredUseCase{repo: repo}
}

func (uc *ListByNeuteredUseCase) Execute(ctx context.Context, input ListByNeuteredInput) (ListByNeuteredOutput, error) {
	dogs, err := uc.repo.ListByNeutered(ctx, input.Neutered(), input.Limit(), input.Offset())
	if err != nil {
		return ListByNeuteredOutput{}, fmt.Errorf("list by neutered: %w", err)
	}
	return ListByNeuteredOutput{Dogs: dogs}, nil
}

// ListByHeatInput is the validated request for listing dogs by the
// heat flag.
type ListByHeatInput struct {
	heat   bool
	limit  int
	offset int
}

func (in ListByHeatInput) Heat() bool  { return in.heat }
func (in ListByHeatInput) Limit() int  { return in.limit }
func (in ListByHeatInput) Offset() int { return in.offset }

func NewListByHeatInput(heat bool, limit, offset int) (ListByHeatInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListByHeatInput{heat: heat, limit: limit, offset: offset}, nil
}

func MustNewListByHeatInput(heat bool, limit, offset int) ListByHeatInput {
	in, err := NewListByHeatInput(heat, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListByHeatOutput struct {
	Dogs []*domain.Dog
}

type ListByHeatUseCase struct {
	repo domain.DogRepository
}

func NewListByHeatUseCase(repo domain.DogRepository) *ListByHeatUseCase {
	return &ListByHeatUseCase{repo: repo}
}

func (uc *ListByHeatUseCase) Execute(ctx context.Context, input ListByHeatInput) (ListByHeatOutput, error) {
	dogs, err := uc.repo.ListByHeat(ctx, input.Heat(), input.Limit(), input.Offset())
	if err != nil {
		return ListByHeatOutput{}, fmt.Errorf("list by heat: %w", err)
	}
	return ListByHeatOutput{Dogs: dogs}, nil
}

// ListByIsActiveInput is the validated request for listing dogs by
// the is_active flag.
type ListByIsActiveInput struct {
	isActive bool
	limit    int
	offset   int
}

func (in ListByIsActiveInput) IsActive() bool { return in.isActive }
func (in ListByIsActiveInput) Limit() int     { return in.limit }
func (in ListByIsActiveInput) Offset() int    { return in.offset }

func NewListByIsActiveInput(isActive bool, limit, offset int) (ListByIsActiveInput, error) {
	limit, offset = normalizePagination(limit, offset)
	return ListByIsActiveInput{isActive: isActive, limit: limit, offset: offset}, nil
}

func MustNewListByIsActiveInput(isActive bool, limit, offset int) ListByIsActiveInput {
	in, err := NewListByIsActiveInput(isActive, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListByIsActiveOutput struct {
	Dogs []*domain.Dog
}

type ListByIsActiveUseCase struct {
	repo domain.DogRepository
}

func NewListByIsActiveUseCase(repo domain.DogRepository) *ListByIsActiveUseCase {
	return &ListByIsActiveUseCase{repo: repo}
}

func (uc *ListByIsActiveUseCase) Execute(ctx context.Context, input ListByIsActiveInput) (ListByIsActiveOutput, error) {
	dogs, err := uc.repo.ListByIsActive(ctx, input.IsActive(), input.Limit(), input.Offset())
	if err != nil {
		return ListByIsActiveOutput{}, fmt.Errorf("list by is_active: %w", err)
	}
	return ListByIsActiveOutput{Dogs: dogs}, nil
}

// ListByAgeBracketInput is the validated request for listing dogs by
// age bracket.
type ListByAgeBracketInput struct {
	ageBracket domain.AgeBracket
	limit      int
	offset     int
}

func (in ListByAgeBracketInput) AgeBracket() domain.AgeBracket { return in.ageBracket }
func (in ListByAgeBracketInput) Limit() int                    { return in.limit }
func (in ListByAgeBracketInput) Offset() int                   { return in.offset }

func NewListByAgeBracketInput(ageBracket domain.AgeBracket, limit, offset int) (ListByAgeBracketInput, error) {
	if !ageBracket.IsValid() {
		return ListByAgeBracketInput{}, &ValidationError{Field: "age_bracket"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListByAgeBracketInput{ageBracket: ageBracket, limit: limit, offset: offset}, nil
}

func MustNewListByAgeBracketInput(ageBracket domain.AgeBracket, limit, offset int) ListByAgeBracketInput {
	in, err := NewListByAgeBracketInput(ageBracket, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListByAgeBracketOutput struct {
	Dogs []*domain.Dog
}

type ListByAgeBracketUseCase struct {
	repo domain.DogRepository
}

func NewListByAgeBracketUseCase(repo domain.DogRepository) *ListByAgeBracketUseCase {
	return &ListByAgeBracketUseCase{repo: repo}
}

func (uc *ListByAgeBracketUseCase) Execute(ctx context.Context, input ListByAgeBracketInput) (ListByAgeBracketOutput, error) {
	dogs, err := uc.repo.ListByAgeBracket(ctx, input.AgeBracket(), input.Limit(), input.Offset())
	if err != nil {
		return ListByAgeBracketOutput{}, fmt.Errorf("list by age bracket: %w", err)
	}
	return ListByAgeBracketOutput{Dogs: dogs}, nil
}

// ListBySizeBracketInput is the validated request for listing dogs by
// size bracket.
type ListBySizeBracketInput struct {
	sizeBracket domain.SizeBracket
	limit       int
	offset      int
}

func (in ListBySizeBracketInput) SizeBracket() domain.SizeBracket { return in.sizeBracket }
func (in ListBySizeBracketInput) Limit() int                      { return in.limit }
func (in ListBySizeBracketInput) Offset() int                     { return in.offset }

func NewListBySizeBracketInput(sizeBracket domain.SizeBracket, limit, offset int) (ListBySizeBracketInput, error) {
	if !sizeBracket.IsValid() {
		return ListBySizeBracketInput{}, &ValidationError{Field: "size_bracket"}
	}
	limit, offset = normalizePagination(limit, offset)
	return ListBySizeBracketInput{sizeBracket: sizeBracket, limit: limit, offset: offset}, nil
}

func MustNewListBySizeBracketInput(sizeBracket domain.SizeBracket, limit, offset int) ListBySizeBracketInput {
	in, err := NewListBySizeBracketInput(sizeBracket, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

type ListBySizeBracketOutput struct {
	Dogs []*domain.Dog
}

type ListBySizeBracketUseCase struct {
	repo domain.DogRepository
}

func NewListBySizeBracketUseCase(repo domain.DogRepository) *ListBySizeBracketUseCase {
	return &ListBySizeBracketUseCase{repo: repo}
}

func (uc *ListBySizeBracketUseCase) Execute(ctx context.Context, input ListBySizeBracketInput) (ListBySizeBracketOutput, error) {
	dogs, err := uc.repo.ListBySizeBracket(ctx, input.SizeBracket(), input.Limit(), input.Offset())
	if err != nil {
		return ListBySizeBracketOutput{}, fmt.Errorf("list by size bracket: %w", err)
	}
	return ListBySizeBracketOutput{Dogs: dogs}, nil
}

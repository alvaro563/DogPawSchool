package dog

import (
	"context"
	"fmt"

	"dogpaw/internal/domain"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 100
)

func NormalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

type ListByOwnerInput struct {
	OwnerID int
	Limit   int
	Offset  int
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

func (uc *ListByOwnerUseCase) Execute(ctx context.Context, in ListByOwnerInput) (ListByOwnerOutput, error) {
	if err := in.validate(); err != nil {
		return ListByOwnerOutput{}, err
	}
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListByOwner(ctx, in.OwnerID, limit, offset)
	if err != nil {
		return ListByOwnerOutput{}, fmt.Errorf("list dogs by owner: %w", err)
	}
	return ListByOwnerOutput{Dogs: dogs}, nil
}

func (in ListByOwnerInput) validate() error {
	if in.OwnerID <= 0 {
		return &ValidationError{Field: "owner_id"}
	}
	return nil
}

type ListAllDogsInput struct {
	Limit  int
	Offset int
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

func (uc *ListAllDogsUseCase) Execute(ctx context.Context, in ListAllDogsInput) (ListAllDogsOutput, error) {
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListAll(ctx, false, limit, offset)
	if err != nil {
		return ListAllDogsOutput{}, fmt.Errorf("list all dogs: %w", err)
	}
	return ListAllDogsOutput{Dogs: dogs}, nil
}

type ListActiveDogsInput struct {
	Limit  int
	Offset int
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

func (uc *ListActiveDogsUseCase) Execute(ctx context.Context, in ListActiveDogsInput) (ListActiveDogsOutput, error) {
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListAll(ctx, true, limit, offset)
	if err != nil {
		return ListActiveDogsOutput{}, fmt.Errorf("list active dogs: %w", err)
	}
	return ListActiveDogsOutput{Dogs: dogs}, nil
}

type ListByIncompatibilityInput struct {
	IncompatibilityID int
	Limit             int
	Offset            int
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

func (uc *ListByIncompatibilityUseCase) Execute(ctx context.Context, in ListByIncompatibilityInput) (ListByIncompatibilityOutput, error) {
	if err := in.validate(); err != nil {
		return ListByIncompatibilityOutput{}, err
	}
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListByIncompatibility(ctx, in.IncompatibilityID, limit, offset)
	if err != nil {
		return ListByIncompatibilityOutput{}, fmt.Errorf("list by incompatibility: %w", err)
	}
	return ListByIncompatibilityOutput{Dogs: dogs}, nil
}

func (in ListByIncompatibilityInput) validate() error {
	if in.IncompatibilityID <= 0 {
		return &ValidationError{Field: "incompatibility_id"}
	}
	return nil
}

type ListByBreedInput struct {
	Breed  string
	Limit  int
	Offset int
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

func (uc *ListByBreedUseCase) Execute(ctx context.Context, in ListByBreedInput) (ListByBreedOutput, error) {
	if err := in.validate(); err != nil {
		return ListByBreedOutput{}, err
	}
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListByBreed(ctx, in.Breed, limit, offset)
	if err != nil {
		return ListByBreedOutput{}, fmt.Errorf("list by breed: %w", err)
	}
	return ListByBreedOutput{Dogs: dogs}, nil
}

func (in ListByBreedInput) validate() error {
	if in.Breed == "" {
		return &ValidationError{Field: "breed"}
	}
	return nil
}

type ListBySexInput struct {
	Sex    domain.Sex
	Limit  int
	Offset int
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

func (uc *ListBySexUseCase) Execute(ctx context.Context, in ListBySexInput) (ListBySexOutput, error) {
	if err := in.validate(); err != nil {
		return ListBySexOutput{}, err
	}
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListBySex(ctx, in.Sex, limit, offset)
	if err != nil {
		return ListBySexOutput{}, fmt.Errorf("list by sex: %w", err)
	}
	return ListBySexOutput{Dogs: dogs}, nil
}

func (in ListBySexInput) validate() error {
	if !in.Sex.IsValid() {
		return &ValidationError{Field: "sex"}
	}
	return nil
}

type ListByNeuteredInput struct {
	Neutered bool
	Limit    int
	Offset   int
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

func (uc *ListByNeuteredUseCase) Execute(ctx context.Context, in ListByNeuteredInput) (ListByNeuteredOutput, error) {
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListByNeutered(ctx, in.Neutered, limit, offset)
	if err != nil {
		return ListByNeuteredOutput{}, fmt.Errorf("list by neutered: %w", err)
	}
	return ListByNeuteredOutput{Dogs: dogs}, nil
}

type ListByHeatInput struct {
	Heat   bool
	Limit  int
	Offset int
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

func (uc *ListByHeatUseCase) Execute(ctx context.Context, in ListByHeatInput) (ListByHeatOutput, error) {
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListByHeat(ctx, in.Heat, limit, offset)
	if err != nil {
		return ListByHeatOutput{}, fmt.Errorf("list by heat: %w", err)
	}
	return ListByHeatOutput{Dogs: dogs}, nil
}

type ListByAgeBracketInput struct {
	AgeBracket domain.AgeBracket
	Limit      int
	Offset     int
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

func (uc *ListByAgeBracketUseCase) Execute(ctx context.Context, in ListByAgeBracketInput) (ListByAgeBracketOutput, error) {
	if err := in.validate(); err != nil {
		return ListByAgeBracketOutput{}, err
	}
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListByAgeBracket(ctx, in.AgeBracket, limit, offset)
	if err != nil {
		return ListByAgeBracketOutput{}, fmt.Errorf("list by age bracket: %w", err)
	}
	return ListByAgeBracketOutput{Dogs: dogs}, nil
}

func (in ListByAgeBracketInput) validate() error {
	if !in.AgeBracket.IsValid() {
		return &ValidationError{Field: "age_bracket"}
	}
	return nil
}

type ListBySizeBracketInput struct {
	SizeBracket domain.SizeBracket
	Limit       int
	Offset      int
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

func (uc *ListBySizeBracketUseCase) Execute(ctx context.Context, in ListBySizeBracketInput) (ListBySizeBracketOutput, error) {
	if err := in.validate(); err != nil {
		return ListBySizeBracketOutput{}, err
	}
	limit, offset := NormalizePagination(in.Limit, in.Offset)
	dogs, err := uc.repo.ListBySizeBracket(ctx, in.SizeBracket, limit, offset)
	if err != nil {
		return ListBySizeBracketOutput{}, fmt.Errorf("list by size bracket: %w", err)
	}
	return ListBySizeBracketOutput{Dogs: dogs}, nil
}

func (in ListBySizeBracketInput) validate() error {
	if !in.SizeBracket.IsValid() {
		return &ValidationError{Field: "size_bracket"}
	}
	return nil
}

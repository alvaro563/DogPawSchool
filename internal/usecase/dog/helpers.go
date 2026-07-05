package dog

import "dogpaw/internal/domain"

func isKnownIncompatibility(s string) bool {
	switch domain.Incompatibility(s) {
	case domain.IncompatibilityReactivoMachos,
		domain.IncompatibilityNoToleraCachorros:
		return true
	}
	return false
}

func containsIncompatibility(list []domain.Incompatibility, target domain.Incompatibility) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func removeIncompatibility(list []domain.Incompatibility, target domain.Incompatibility) []domain.Incompatibility {
	out := make([]domain.Incompatibility, 0, len(list))
	for _, v := range list {
		if v != target {
			out = append(out, v)
		}
	}
	return out
}

func validateIncompatibilityInput(dogID int, incompat string) error {
	if dogID <= 0 {
		return &ValidationError{Field: "dog_id"}
	}
	if incompat == "" {
		return &ValidationError{Field: "incompatibility"}
	}
	if !isKnownIncompatibility(incompat) {
		return &ValidationError{Field: "incompatibility"}
	}
	return nil
}

package dog

const (
	defaultPageLimit = 50
	maxPageLimit     = 100
)

// normalizePagination clamps a (limit, offset) pair to the bounds
// used by every list input in this package. A non-positive limit
// falls back to defaultPageLimit, a limit above maxPageLimit is
// capped, and a negative offset is reset to 0. Pagination
// normalization is the only place where these rules live: every
// list input factory calls this so the use case can trust the
// values it gets.
func normalizePagination(limit, offset int) (int, int) {
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

// validateTwoIDs is the shared validation rule for the add / remove
// dog-incompatibility inputs. Returned *ValidationError mirrors the
// per-field rules of those factories.
func validateTwoIDs(dogID, incompatibilityID int) error {
	if dogID <= 0 {
		return &ValidationError{Field: "dog_id"}
	}
	if incompatibilityID <= 0 {
		return &ValidationError{Field: "incompatibility_id"}
	}
	return nil
}

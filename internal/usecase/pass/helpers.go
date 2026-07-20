package pass

const (
	defaultPageLimit = 50
	maxPageLimit     = 100
)

// NormalizePagination clamps the requested limit/offset to safe
// defaults. A non-positive limit falls back to defaultPageLimit, a
// limit above maxPageLimit is capped, and a negative offset is reset
// to 0.
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

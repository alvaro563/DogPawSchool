package reservation

// normalizePagination clamps a (limit, offset) pair to the bounds
// used by every list input in this package. A non-positive limit
// falls back to 50, a limit above 100 is capped, and a negative
// offset is reset to 0. Pagination normalization is the only place
// where these rules live: every list input factory calls this so
// the use case can trust the values it gets.
func normalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

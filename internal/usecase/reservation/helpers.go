package reservation

// NormalizePagination clamps a (limit, offset) pair to the bounds
// used by every list endpoint in this module: limit defaults to
// 50, offset to 0, limit is capped at 100. The same rules are
// used by the pass and activity packages; the duplication is
// tracked as a follow-up to extract a shared package.
//
// Negative inputs are treated as the default (the standard "missing
// query param" semantics for an Atoi that returns 0 on empty).
func NormalizePagination(limit, offset int) (int, int) {
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

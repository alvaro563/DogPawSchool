package reservation

import (
	"context"
	"fmt"
	"time"

	"dogpaw/internal/domain"
)

// Attendance-report pagination caps.
//
// Two distinct caps exist because the report is consumed in two very
// different ways:
//
//   - The UI table is paginated and prefers bounded rows (cap 1000).
//     The admin can drill down with offset/limit to scroll through
//     the full history if needed.
//
//   - The CSV export is meant to be a complete dump for an external
//     spreadsheet. Capping it at the UI cap would silently truncate
//     year-long reports. We allow up to 50_000 rows per export; at
//     ~12 dogs/activity and ~2 activities/day that is roughly six
//     years of data, which is what an admin realistically needs.
const (
	AttendanceReportDefaultLimit   = 100
	AttendanceReportMaxLimitUI     = 1000
	AttendanceReportMaxLimitExport = 50000
)

// ListAttendanceReportInput is the validated input for the admin
// attendance report. limit/offset are normalized by the factories.
// from / to are both optional: nil = no bound on that side. When
// both are non-nil, the report covers [from, to] inclusive on both
// ends (so a single-day export with from == to returns that day's
// COMPLETED rows).
type ListAttendanceReportInput struct {
	from   *time.Time
	to     *time.Time
	limit  int
	offset int
}

func (in ListAttendanceReportInput) From() *time.Time { return in.from }
func (in ListAttendanceReportInput) To() *time.Time   { return in.to }
func (in ListAttendanceReportInput) Limit() int       { return in.limit }
func (in ListAttendanceReportInput) Offset() int      { return in.offset }

// NewListAttendanceReportInput is the factory for the UI export of
// the attendance report. limit 0 → default; offset 0 → start.
// Errors when: limit < 0, limit > UI cap, offset < 0, from > to.
// Validation errors are typed so the handler can map them to a 400
// response with the precise field name.
func NewListAttendanceReportInput(from, to *time.Time, limit, offset int) (ListAttendanceReportInput, error) {
	if limit == 0 {
		limit = AttendanceReportDefaultLimit
	}
	if limit < 0 {
		return ListAttendanceReportInput{}, &ValidationError{Field: "limit"}
	}
	if limit > AttendanceReportMaxLimitUI {
		return ListAttendanceReportInput{}, &ValidationError{Field: "limit"}
	}
	if offset < 0 {
		return ListAttendanceReportInput{}, &ValidationError{Field: "offset"}
	}
	if from != nil && to != nil && from.After(*to) {
		return ListAttendanceReportInput{}, &ValidationError{Field: "to"}
	}
	return ListAttendanceReportInput{
		from:   from,
		to:     to,
		limit:  limit,
		offset: offset,
	}, nil
}

// NewExportAttendanceReportInput is the factory for the CSV export.
// It uses the export cap (50_000), always offset 0, and accepts
// from == to (single-day report).
func NewExportAttendanceReportInput(from, to *time.Time) (ListAttendanceReportInput, error) {
	if from != nil && to != nil && from.After(*to) {
		return ListAttendanceReportInput{}, &ValidationError{Field: "to"}
	}
	return ListAttendanceReportInput{
		from:   from,
		to:     to,
		limit:  AttendanceReportMaxLimitExport,
		offset: 0,
	}, nil
}

// MustNewListAttendanceReportInput panics on error. For tests.
func MustNewListAttendanceReportInput(from, to *time.Time, limit, offset int) ListAttendanceReportInput {
	in, err := NewListAttendanceReportInput(from, to, limit, offset)
	if err != nil {
		panic(err)
	}
	return in
}

// MustNewExportAttendanceReportInput panics on error. For tests.
func MustNewExportAttendanceReportInput(from, to *time.Time) ListAttendanceReportInput {
	in, err := NewExportAttendanceReportInput(from, to)
	if err != nil {
		panic(err)
	}
	return in
}

// ListAttendanceReportOutput carries the read-model entries plus the
// effective pagination values so the handler can build a clean
// response envelope without re-parsing query strings.
type ListAttendanceReportOutput struct {
	Entries []*domain.AttendanceReportEntry
	Limit   int
	Offset  int
}

// ListAttendanceReportUseCase executes the admin attendance report.
// Both the JSON endpoint and the CSV endpoint go through this same
// use case to keep the read-model definition in one place.
type ListAttendanceReportUseCase struct {
	repo domain.ReservationRepository
}

func NewListAttendanceReportUseCase(repo domain.ReservationRepository) *ListAttendanceReportUseCase {
	return &ListAttendanceReportUseCase{repo: repo}
}

// Execute runs the report. The repo applies the (status=COMPLETED +
// from/to) predicate.
func (uc *ListAttendanceReportUseCase) Execute(ctx context.Context, in ListAttendanceReportInput) (ListAttendanceReportOutput, error) {
	entries, err := uc.repo.ListAttendanceReport(ctx, in.from, in.to, in.limit, in.offset)
	if err != nil {
		return ListAttendanceReportOutput{}, fmt.Errorf("list attendance report: %w", err)
	}
	return ListAttendanceReportOutput{
		Entries: entries,
		Limit:   in.limit,
		Offset:  in.offset,
	}, nil
}

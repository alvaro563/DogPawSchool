package handler

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	reservationuc "dogpaw/internal/usecase/reservation"
)

// AttendanceReportLister is the contract the handler depends on for
// both the JSON and the CSV endpoints. The CSV path reuses the same
// use case; only the serialization differs.
type AttendanceReportLister interface {
	Execute(ctx context.Context, in reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error)
}

// AttendanceHandler serves the admin attendance report.
//
// Two endpoints share the same use case:
//
//	GET /api/v1/reservations/attendance            → JSON
//	GET /api/v1/reservations/attendance.csv        → CSV download
//
// Both accept the same query params (?from=, ?to=, ?limit=, ?offset=).
// Hard-coded to status = COMPLETED; this is what "qué perros han
// participado" means in business terms.
type AttendanceHandler struct {
	lister AttendanceReportLister
}

func NewAttendanceHandler(lister AttendanceReportLister) *AttendanceHandler {
	return &AttendanceHandler{lister: lister}
}

// attendanceResponse is the JSON envelope for the admin table. The
// "count" field is the size of the returned slice (NOT a global
// total) so the UI can show "showing N rows" without a second call.
type attendanceResponse struct {
	Entries []attendanceEntryDTO `json:"entries"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
	Count   int                  `json:"count"`
}

// attendanceEntryDTO mirrors domain.AttendanceReportEntry in JSON.
// Snake_case to match every other response in the API.
type attendanceEntryDTO struct {
	ReservationID int       `json:"reservation_id"`
	ActivityID    int       `json:"activity_id"`
	ActivityName  string    `json:"activity_name"`
	ActivityDate  time.Time `json:"activity_date"`
	DogID         int       `json:"dog_id"`
	DogName       string    `json:"dog_name"`
	DogPassport   string    `json:"dog_passport"`
}

func toAttendanceResponse(out reservationuc.ListAttendanceReportOutput) attendanceResponse {
	entries := make([]attendanceEntryDTO, 0, len(out.Entries))
	for _, e := range out.Entries {
		entries = append(entries, attendanceEntryDTO{
			ReservationID: e.ReservationID,
			ActivityID:    e.ActivityID,
			ActivityName:  e.ActivityName,
			ActivityDate:  e.ActivityDate,
			DogID:         e.DogID,
			DogName:       e.DogName,
			DogPassport:   e.DogPassport,
		})
	}
	return attendanceResponse{
		Entries: entries,
		Limit:   out.Limit,
		Offset:  out.Offset,
		Count:   len(entries),
	}
}

// List godoc
// @Summary      Attendance report (JSON)
// @Description  Returns all COMPLETED reservations whose activity date is in the optional [from, to] range. Both bounds are RFC3339 timestamps; "to" is inclusive. Date filtering targets the *activity* date (not the booking date). Pagination defaults to 100 rows, capped at 1000 for this endpoint. Status is fixed to COMPLETED.
// @Tags         attendance
// @Produce      json
// @Param        from   query string false "Activity date lower bound, RFC3339. Inclusive."
// @Param        to     query string false "Activity date upper bound, RFC3339. Inclusive."
// @Param        limit  query int    false "Max rows (default 100, max 1000)"
// @Param        offset query int    false "Rows to skip (default 0)"
// @Success      200 {object} attendanceResponse
// @Failure      400 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Security     BearerAuth
// @Router       /api/v1/reservations/attendance [get]
func (h *AttendanceHandler) List(c *gin.Context) {
	from, to, validationErr := parseAttendanceFromTo(c)
	if validationErr != nil {
		c.JSON(http.StatusBadRequest, validationErr.body())
		return
	}
	limit, offset := parseAttendanceLimitOffset(c)

	in, err := reservationuc.NewListAttendanceReportInput(from, to, limit, offset)
	if err != nil {
		writeError(c, err)
		return
	}

	out, err := h.lister.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAttendanceResponse(out))
}

// DownloadCSV godoc
// @Summary      Attendance report (CSV download)
// @Description  Same query as the JSON endpoint, but returns a CSV file (UTF-8 with BOM, RFC 4180) and uses a higher row cap (50_000) so a year-long report is never truncated. The first row is the Spanish header. Status is fixed to COMPLETED.
// @Tags         attendance
// @Produce      text/csv
// @Param        from query string false "Activity date lower bound, RFC3339. Inclusive."
// @Param        to   query string false "Activity date upper bound, RFC3339. Inclusive."
// @Success      200 {file} file "text/csv"
// @Failure      400 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Security     BearerAuth
// @Router       /api/v1/reservations/attendance.csv [get]
func (h *AttendanceHandler) DownloadCSV(c *gin.Context) {
	from, to, validationErr := parseAttendanceFromTo(c)
	if validationErr != nil {
		c.JSON(http.StatusBadRequest, validationErr.body())
		return
	}

	in, err := reservationuc.NewExportAttendanceReportInput(from, to)
	if err != nil {
		writeError(c, err)
		return
	}

	out, err := h.lister.Execute(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}

	var buf bytes.Buffer
	// UTF-8 BOM so Excel Windows opens the file with correct encoding.
	buf.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"ID Reserva", "ID Actividad", "Actividad", "Fecha",
		"ID Perro", "Perro", "Pasaporte",
	})
	for _, e := range out.Entries {
		_ = w.Write([]string{
			strconv.Itoa(e.ReservationID),
			strconv.Itoa(e.ActivityID),
			e.ActivityName,
			e.ActivityDate.UTC().Format(time.RFC3339),
			strconv.Itoa(e.DogID),
			e.DogName,
			e.DogPassport,
		})
	}
	w.Flush()

	filename := buildAttendanceFilename(from, to)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	// gin.ResponseWriter implements io.Writer; copy writes the buffered CSV directly to the wire.
	_, _ = c.Writer.Write(buf.Bytes())
}

// attendanceFieldErr carries the field name + details for the
// "invalid date in ?from= or ?to=" validation.
type attendanceFieldErr struct {
	field   string
	details string
}

func (e attendanceFieldErr) body() errorResponse {
	return errorResponse{Error: "validation", Field: e.field, Details: e.details}
}

// parseAttendanceFromTo extracts the optional ?from= / ?to= query
// params and parses them as RFC3339. Returns an attendanceFieldErr
// (typed) on parse failure so the caller can surface a precise 400.
func parseAttendanceFromTo(c *gin.Context) (*time.Time, *time.Time, *attendanceFieldErr) {
	var from, to *time.Time
	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, nil, &attendanceFieldErr{field: "from", details: "invalid RFC3339 date"}
		}
		from = &t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, nil, &attendanceFieldErr{field: "to", details: "invalid RFC3339 date"}
		}
		to = &t
	}
	return from, to, nil
}

// parseAttendanceLimitOffset follows the convention used by every
// other handler in this package: Atoi ignores parse errors and
// returns 0, which the factory normalises to the default.
func parseAttendanceLimitOffset(c *gin.Context) (limit, offset int) {
	limit, _ = strconv.Atoi(c.Query("limit"))
	offset, _ = strconv.Atoi(c.Query("offset"))
	return limit, offset
}

// buildAttendanceFilename produces the suggested download filename.
// When both dates are absent, the suffix is "all" to make it clear.
func buildAttendanceFilename(from, to *time.Time) string {
	fromSlug := "all"
	toSlug := "all"
	if from != nil {
		fromSlug = from.UTC().Format("2006-01-02")
	}
	if to != nil {
		toSlug = to.UTC().Format("2006-01-02")
	}
	return fmt.Sprintf("asistencia_%s_%s.csv", fromSlug, toSlug)
}

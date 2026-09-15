package handler

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
	reservationuc "dogpaw/internal/usecase/reservation"
)

// ----------------------------------------------------------------------------
// Stub
// ----------------------------------------------------------------------------

type stubAttendanceLister struct {
	fn func(ctx context.Context, in reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error)
}

func (s *stubAttendanceLister) Execute(ctx context.Context, in reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
	return s.fn(ctx, in)
}

func newAttendanceHandler(stub *stubAttendanceLister) *AttendanceHandler {
	return &AttendanceHandler{lister: stub}
}

// ----------------------------------------------------------------------------
// JSON endpoint
// ----------------------------------------------------------------------------

func TestAttendanceList_Success_Default(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	entries := []*domain.AttendanceReportEntry{
		{
			ReservationID: 1, ActivityID: 10, ActivityName: "Paseo Centro",
			ActivityDate: day, DogID: 5, DogName: "Luna", DogPassport: "ES-1001",
		},
	}
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(_ context.Context, in reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		assert.Nil(t, in.From())
		assert.Nil(t, in.To())
		assert.Equal(t, reservationuc.AttendanceReportDefaultLimit, in.Limit())
		assert.Equal(t, 0, in.Offset())
		return reservationuc.ListAttendanceReportOutput{
			Entries: entries,
			Limit:   in.Limit(),
			Offset:  in.Offset(),
		}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance", "")
	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	body := w.Body.String()
	assert.Contains(t, body, `"reservation_id":1`)
	assert.Contains(t, body, `"activity_name":"Paseo Centro"`)
	assert.Contains(t, body, `"dog_passport":"ES-1001"`)
	assert.Contains(t, body, `"count":1`)
}

func TestAttendanceList_WithDateRange(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(_ context.Context, in reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		require.NotNil(t, in.From())
		require.NotNil(t, in.To())
		assert.Equal(t, from, *in.From())
		assert.Equal(t, to, *in.To())
		return reservationuc.ListAttendanceReportOutput{Entries: nil, Limit: 100, Offset: 0}, nil
	}})
	c, w := setupCtx(http.MethodGet,
		"/api/v1/reservations/attendance?from=2026-02-01T00:00:00Z&to=2026-02-28T23:59:59Z&limit=50&offset=10", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAttendanceList_InvalidLimitCap(t *testing.T) {
	t.Parallel()
	stub := &stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		t.Fatal("use case must not be called for invalid input")
		return reservationuc.ListAttendanceReportOutput{}, nil
	}}
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance?limit=9999", "")
	newAttendanceHandler(stub).List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "limit")
}

func TestAttendanceList_NegativeOffset(t *testing.T) {
	t.Parallel()
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance?offset=-5", "")
	newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		t.Fatal("use case must not be called for invalid input")
		return reservationuc.ListAttendanceReportOutput{}, nil
	}}).List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "offset")
}

func TestAttendanceList_BadFromFormat(t *testing.T) {
	t.Parallel()
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance?from=not-a-date", "")
	newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		t.Fatal("use case must not be called for invalid input")
		return reservationuc.ListAttendanceReportOutput{}, nil
	}}).List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"from"`)
}

func TestAttendanceList_BadToFormat(t *testing.T) {
	t.Parallel()
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance?to=yesterday", "")
	newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		t.Fatal("use case must not be called for invalid input")
		return reservationuc.ListAttendanceReportOutput{}, nil
	}}).List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"to"`)
}

func TestAttendanceList_FromAfterTo(t *testing.T) {
	t.Parallel()
	c, w := setupCtx(http.MethodGet,
		"/api/v1/reservations/attendance?from=2026-03-01T00:00:00Z&to=2026-02-01T00:00:00Z", "")
	newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		t.Fatal("use case must not be called when from > to")
		return reservationuc.ListAttendanceReportOutput{}, nil
	}}).List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"to"`)
}

func TestAttendanceList_RepoError(t *testing.T) {
	t.Parallel()
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		return reservationuc.ListAttendanceReportOutput{}, errors.New("db down")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance", "")
	h.List(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAttendanceList_EmptyResult(t *testing.T) {
	t.Parallel()
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		return reservationuc.ListAttendanceReportOutput{Entries: nil, Limit: 100, Offset: 0}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance", "")
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"entries":[]`)
	assert.Contains(t, body, `"count":0`)
}

// ----------------------------------------------------------------------------
// CSV endpoint
// ----------------------------------------------------------------------------

func TestAttendanceDownloadCSV_Success(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(_ context.Context, in reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		// CSV factory sets limit=AttendanceReportMaxLimitExport and offset=0.
		assert.Equal(t, reservationuc.AttendanceReportMaxLimitExport, in.Limit())
		assert.Equal(t, 0, in.Offset())
		return reservationuc.ListAttendanceReportOutput{
			Entries: []*domain.AttendanceReportEntry{
				{
					ReservationID: 412, ActivityID: 42,
					ActivityName: "Paseo, con coma", ActivityDate: day,
					DogID: 7, DogName: "Luna \"estrellita\"", DogPassport: "ES-12345",
				},
				{
					ReservationID: 413, ActivityID: 43,
					ActivityName: "Clase Mañana", ActivityDate: day.Add(24 * time.Hour),
					DogID: 8, DogName: "Toby", DogPassport: "ES-67890",
				},
			},
			Limit:  in.Limit(),
			Offset: 0,
		}, nil
	}})
	c, w := setupCtx(http.MethodGet,
		"/api/v1/reservations/attendance.csv?from=2026-02-01T00:00:00Z&to=2026-02-28T23:59:59Z", "")
	h.DownloadCSV(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	disp := w.Header().Get("Content-Disposition")
	assert.True(t, strings.HasPrefix(disp, "attachment; filename="), "expected attachment header, got %q", disp)
	assert.Contains(t, disp, "asistencia_2026-02-01_2026-02-28.csv")

	body := w.Body.Bytes()
	// First three bytes must be the UTF-8 BOM.
	assert.Equal(t, byte(0xEF), body[0])
	assert.Equal(t, byte(0xBB), body[1])
	assert.Equal(t, byte(0xBF), body[2])

	// Parse the BOM-stripped body as RFC 4180 CSV and verify the
	// header row plus the two data rows, including quoted fields with
	// commas and escaped quotes.
	rdr := csv.NewReader(strings.NewReader(string(body[3:])))
	rows, err := rdr.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, []string{"ID Reserva", "ID Actividad", "Actividad", "Fecha", "ID Perro", "Perro", "Pasaporte"}, rows[0])
	assert.Equal(t, []string{"412", "42", "Paseo, con coma", "2026-02-15T10:00:00Z", "7", "Luna \"estrellita\"", "ES-12345"}, rows[1])
	assert.Equal(t, []string{"413", "43", "Clase Mañana", "2026-02-16T10:00:00Z", "8", "Toby", "ES-67890"}, rows[2])
}

func TestAttendanceDownloadCSV_Empty(t *testing.T) {
	t.Parallel()
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		return reservationuc.ListAttendanceReportOutput{Entries: nil, Limit: reservationuc.AttendanceReportMaxLimitExport, Offset: 0}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance.csv", "")
	h.DownloadCSV(c)
	require.Equal(t, http.StatusOK, w.Code)

	// BOM + header row only.
	body := w.Body.Bytes()
	stripped := body[3:]
	rdr := csv.NewReader(strings.NewReader(string(stripped)))
	rows, err := rdr.ReadAll()
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestAttendanceDownloadCSV_BadFrom(t *testing.T) {
	t.Parallel()
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		t.Fatal("use case must not be called for invalid input")
		return reservationuc.ListAttendanceReportOutput{}, nil
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance.csv?from=not-a-date", "")
	h.DownloadCSV(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"field":"from"`)
}

func TestAttendanceDownloadCSV_RepoError(t *testing.T) {
	t.Parallel()
	h := newAttendanceHandler(&stubAttendanceLister{fn: func(context.Context, reservationuc.ListAttendanceReportInput) (reservationuc.ListAttendanceReportOutput, error) {
		return reservationuc.ListAttendanceReportOutput{}, errors.New("disk full")
	}})
	c, w := setupCtx(http.MethodGet, "/api/v1/reservations/attendance.csv", "")
	h.DownloadCSV(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAttendanceResponseEnvelope exercises the JSON envelope shape so
// regressions in toAttendanceResponse are caught early.
func TestAttendanceResponseEnvelope(t *testing.T) {
	t.Parallel()
	out := reservationuc.ListAttendanceReportOutput{
		Entries: []*domain.AttendanceReportEntry{
			{ReservationID: 1, ActivityID: 2, ActivityName: "X", ActivityDate: time.Unix(0, 0), DogID: 3, DogName: "D", DogPassport: "P"},
		},
		Limit: 100, Offset: 0,
	}
	resp := toAttendanceResponse(out)
	assert.Equal(t, 1, resp.Count)
	assert.Equal(t, 100, resp.Limit)
	assert.Equal(t, 0, resp.Offset)
	assert.Len(t, resp.Entries, 1)
	assert.Equal(t, "X", resp.Entries[0].ActivityName)
}

// silence unused io import when no test uses it directly.
var _ = io.Discard

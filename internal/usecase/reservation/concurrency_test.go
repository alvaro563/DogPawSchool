package reservation

import (
	"context"
	"sync"
	"testing"
	"time"

	"dogpaw/internal/domain"
)

// TestRegisterReservation_ConcurrentExecuteIsRaceFree exercises a single
// use case instance from several goroutines, exactly as the HTTP router
// does: newRouter builds one RegisterReservationUseCase and every request
// shares it. Run with -race.
func TestRegisterReservation_ConcurrentExecuteIsRaceFree(t *testing.T) {
	base := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)

	activityRepo := &stubActivityRepository{
		getByID: func(ctx context.Context, id int) (*domain.Activity, error) {
			return domain.MustNewActivity(id, "Ruta", "Parque", domain.TypeRoute, 100, 1,
				base.Add(72*time.Hour)), nil
		},
	}
	dogRepo := &stubDogRepository{
		getByID: func(ctx context.Context, id int) (*domain.Dog, error) {
			return domain.NewDog(id, "Rex", "Mestizo", "ES-1", 24, domain.SexMale, 10, 1)
		},
	}
	passRepo := &stubPassRepository{
		getByID: func(ctx context.Context, id int) (*domain.Pass, error) {
			return domain.MustNewPass(id, 10, 10, 1000, domain.PassGeneric, 1, base, base, nil), nil
		},
	}
	reservationRepo := &mockReservationRepository{
		create: func(ctx context.Context, r *domain.Reservation) (int, error) { return 1, nil },
	}

	uc := NewRegisterReservationUseCase(
		&stubTransactor{}, activityRepo, dogRepo, passRepo, reservationRepo,
	)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			clock := base.Add(time.Duration(i) * time.Second)
			in := MustNewRegisterReservationInput(1, 1, 1, 1, func() time.Time { return clock })
			if _, err := uc.Execute(context.Background(), in); err != nil {
				t.Errorf("execute: %v", err)
			}
		}(i)
	}
	wg.Wait()
}

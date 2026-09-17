import { Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Calendar, Dog, HandHeart, ArrowLeft, AlertCircle } from 'lucide-react';
import { fetchAllReservations } from '@/infrastructure/repositories/reservation-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { ForgiveReservationButton } from '@/features/admin/components/forgive-reservation-button';

// CancelledLateReservationsPage: admin-only view of every
// reservation in StatusCancelledLate. Mirrors the styling of the
// "Actividades" page (espejo) so the admin sees a consistent
// header + row pattern across admin listings.
//
// The page is reachable from the "Ver canceladas tarde (N)" link
// in the Reservas page header. Each row exposes a
// <ForgiveReservationButton /> that transitions the reservation to
// FORGIVEN and refunds the pass session when refundable.
export function CancelledLateReservationsPage() {
  const { data: reservations = [], isLoading, error } = useQuery({
    queryKey: ['admin-reservations', 'cancelled-late'],
    queryFn: () => fetchAllReservations('CANCELLED_LATE'),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  const subtitle =
    reservations.length === 0
      ? 'Cuando un administrador cancele una reserva fuera del plazo de devolución aparecerá aquí para que pueda perdonarla.'
      : reservations.length === 1
        ? '1 reserva fuera de plazo · pulsa "Perdonar y devolver" para devolverle la sesión al bono'
        : `${reservations.length} reservas fuera de plazo · pulsa "Perdonar y devolver" en cada una para devolver la sesión al bono`;

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Reservas canceladas tarde</h1>
          <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
        </div>
        <Link
          to="/admin/reservations"
          className="inline-flex items-center gap-1 self-start rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted sm:self-auto"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Volver a reservas
        </Link>
      </div>

      {error ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-destructive/10">
            <AlertCircle className="h-8 w-8 text-destructive" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">Error al cargar las reservas</p>
        </div>
      ) : reservations.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <HandHeart className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">No hay reservas canceladas tarde</p>
          <p className="text-sm text-muted-foreground">{subtitle}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {reservations.map((r) => (
            <div
              key={r.id}
              className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold">{r.activity_name}</p>
                <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                  <span className="inline-flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    {new Date(r.activity_date).toLocaleDateString('es-ES', {
                      day: 'numeric',
                      month: 'short',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </span>
                  <span>·</span>
                  <span className="inline-flex items-center gap-1">
                    <Dog className="h-3 w-3" />
                    {r.dog_name} (ID: {r.dog_id})
                  </span>
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <span className="rounded-full bg-red-50 px-2.5 py-0.5 text-xs font-medium text-red-600 dark:bg-red-900/20 dark:text-red-400">
                  Cancelada tarde
                </span>
                <ForgiveReservationButton
                  reservationId={r.id}
                  dogName={r.dog_name}
                  onInvalidate={() => {
                    // The button already invalidates the query key.
                  }}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

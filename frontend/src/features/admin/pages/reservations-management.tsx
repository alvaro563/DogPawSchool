import { useQuery } from '@tanstack/react-query';
import { ClipboardList, Dog } from 'lucide-react';
import { fetchUpcomingReservations } from '@/infrastructure/repositories/reservation-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { CancelReservationButton } from '@/features/admin/components/cancel-reservation-button';
import { RejectReservationButton } from '@/features/admin/components/reject-reservation-button';

const STATUS_STYLES: Record<string, string> = {
  CONFIRMED: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300',
  PENDING_TO_CONFIRM: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  COMPLETED: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
  CANCELLED_IN_TIME: 'bg-muted text-muted-foreground',
  CANCELLED_LATE: 'bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400',
  FORGIVEN: 'bg-muted text-muted-foreground',
  NO_SHOW: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
};

const STATUS_LABELS: Record<string, string> = {
  CONFIRMED: 'Confirmada',
  PENDING_TO_CONFIRM: 'Pendiente',
  COMPLETED: 'Completada',
  CANCELLED_IN_TIME: 'Cancelada',
  CANCELLED_LATE: 'Cancelada tarde',
  FORGIVEN: 'Perdonada',
  NO_SHOW: 'No presentado',
};

export function ReservationsManagementPage() {
  const { data: reservations = [], isLoading, error } = useQuery({
    queryKey: ['all-reservations'],
    queryFn: fetchUpcomingReservations,
  });

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-20">
        <p className="text-sm text-destructive">Error al cargar las reservas</p>
        <p className="text-xs text-muted-foreground">Inténtalo de nuevo más tarde</p>
      </div>
    );
  }

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Reservas</h1>
        <p className="mt-1 text-sm text-muted-foreground">{reservations.length} reservas en total</p>
      </div>

      {reservations.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <ClipboardList className="h-10 w-10 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No hay reservas registradas</p>
        </div>
      ) : (
        <div className="space-y-2">
          {reservations.map((r) => (
            <div key={r.id} className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold">{r.activity_name}</p>
                <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                  <span>{new Date(r.activity_date).toLocaleDateString('es-ES', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}</span>
                  <span>·</span>
                  <span className="flex items-center gap-1"><Dog className="h-3 w-3" />{r.dog_name} (ID: {r.dog_id})</span>
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <span className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLES[r.status] || 'bg-muted text-muted-foreground'}`}>
                  {STATUS_LABELS[r.status] || r.status}
                </span>
                {r.status === 'CONFIRMED' && (
                  <CancelReservationButton
                    reservationId={r.id}
                    ownerId={r.owner_id}
                    dogName={r.dog_name}
                    activityName={r.activity_name}
                    activityDate={r.activity_date}
                    isAdmin={true}
                    onInvalidate={() => {}}
                    variant="compact"
                  />
                )}
                {r.status === 'PENDING_TO_CONFIRM' && (
                  <RejectReservationButton
                    reservationId={r.id}
                    ownerId={r.owner_id}
                    dogName={r.dog_name}
                    onInvalidate={() => {}}
                  />
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

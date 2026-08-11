import { useQuery } from '@tanstack/react-query';
import { ClipboardList, Dog, Ticket } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { fetchUserReservations } from '@/infrastructure/repositories/reservation-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import type { ReservationView } from '@/domain/entities/reservation';

const SHOWN_STATUSES = new Set(['CONFIRMED', 'PENDING_TO_CONFIRM']);

function filterActive(reservations: ReservationView[]): ReservationView[] {
  return reservations.filter((r) => SHOWN_STATUSES.has(r.status));
}

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

export function ReservationListPage() {
  const { user } = useAuth();

  const { data: allReservations = [], isLoading, error } = useQuery({
    queryKey: ['reservations', user?.id],
    queryFn: () => fetchUserReservations(user!.id),
    enabled: !!user,
  });

  const reservations = filterActive(allReservations);

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-20">
        <p className="text-sm text-destructive">Error al cargar tus reservas</p>
        <p className="text-xs text-muted-foreground">Inténtalo de nuevo más tarde</p>
      </div>
    );
  }

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Mis Reservas</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {reservations.length} {reservations.length === 1 ? 'reserva' : 'reservas'}
        </p>
      </div>

      {reservations.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <ClipboardList className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">Aún no tienes reservas</p>
          <p className="text-sm text-muted-foreground">Reserva una clase desde el calendario</p>
        </div>
      ) : (
        <div className="space-y-3">
          {reservations.map((r) => (
            <div key={r.id} className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold">{r.activity_name}</p>
                <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                  <span>{new Date(r.activity_date).toLocaleDateString('es-ES', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}</span>
                  <span>·</span>
                  <span className="flex items-center gap-1"><Dog className="h-3 w-3" />{r.dog_name}</span>
                  {r.pass_type && (
                    <>
                      <span>·</span>
                      <span className="flex items-center gap-1"><Ticket className="h-3 w-3" />{r.pass_type === 'GENERICO' ? 'Genérico' : 'Específico'}</span>
                    </>
                  )}
                </div>
              </div>
              <span className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLES[r.status] || 'bg-muted text-muted-foreground'}`}>
                {STATUS_LABELS[r.status] || r.status}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

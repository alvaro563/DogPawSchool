import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Clock, Dog, MapPin, School, User, X, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { fetchPendingReservations } from '@/infrastructure/repositories/reservation-repository.impl';
import { confirmReservation, rejectReservation } from '@/infrastructure/repositories/reservation-repository.impl';
import { useToast } from '@/features/ui/hooks/toast-context';
import type { PendingReservationEntry } from '@/domain/entities/reservation';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

interface PendingRowProps {
  entry: PendingReservationEntry;
  onInvalidate: () => void;
}

// PendingRow renders one pending reservation with inline Confirm /
// Reject buttons. Both buttons invalidate the parent query on
// success so the entry disappears (confirmed entries move out of
// the pending list; rejected ones are removed permanently).
function PendingRow({ entry, onInvalidate }: PendingRowProps) {
  const toast = useToast();
  const [error, setError] = useState('');

  const confirmMutation = useMutation({
    mutationFn: () => confirmReservation(entry.owner_id, entry.reservation_id),
    onSuccess: () => {
      toast.success(
        'Reserva confirmada',
        `${entry.dog_name} queda inscrito en ${entry.activity_name}.`,
      );
      onInvalidate();
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al aprobar la reserva.')),
  });

  const rejectMutation = useMutation({
    mutationFn: () => rejectReservation(entry.owner_id, entry.reservation_id),
    onSuccess: () => {
      toast.warning(
        'Reserva rechazada',
        `${entry.dog_name} ha sido rechazado para ${entry.activity_name}.`,
      );
      onInvalidate();
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al rechazar la reserva.')),
  });

  const pending = confirmMutation.isPending || rejectMutation.isPending;

  return (
    <div>
      <div className="flex items-center justify-between gap-3 rounded-lg border border-amber-200 bg-amber-50/30 p-3 dark:border-amber-900/40 dark:bg-amber-950/10">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-2">
            <Dog className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="truncate text-sm font-semibold">{entry.dog_name}</span>
            <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
              <User className="h-3 w-3" />
              {entry.owner_name || `Usuario #${entry.owner_id}`}
            </span>
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1">
              <School className="h-3 w-3" />
              {entry.activity_name}
            </span>
            <span>·</span>
            <span>
              {new Date(entry.activity_date).toLocaleString('es-ES', {
                weekday: 'short',
                day: 'numeric',
                month: 'short',
                hour: '2-digit',
                minute: '2-digit',
              })}
            </span>
            <span>·</span>
            <span className="inline-flex items-center gap-1">
              <MapPin className="h-3 w-3" />
              {entry.activity_location}
            </span>
          </div>
          {entry.pending_reasons && entry.pending_reasons.length > 0 && (
            <ul className="mt-2 space-y-1 border-t border-amber-200 pt-2 text-xs text-amber-900 dark:border-amber-900/40 dark:text-amber-200">
              {entry.pending_reasons.map((reason, i) => (
                <li key={i} className="flex gap-1.5">
                  <AlertCircle className="mt-0.5 h-3 w-3 shrink-0" />
                  <span>{reason}</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1 border-emerald-300 text-emerald-700 hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/30"
            onClick={() => { setError(''); confirmMutation.mutate(); }}
            disabled={pending}
          >
            {confirmMutation.isPending ? <LoadingSpinner size="sm" /> : <Check className="h-3.5 w-3.5" />}
            <span className="hidden sm:inline">Aprobar</span>
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1 border-red-300 text-red-700 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950/30"
            onClick={() => { setError(''); rejectMutation.mutate(); }}
            disabled={pending}
          >
            {rejectMutation.isPending ? <LoadingSpinner size="sm" /> : <X className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </div>
      {error && (
        <div className="mt-1 flex items-start gap-2 rounded-md bg-destructive/10 px-3 py-1.5 text-xs text-destructive">
          <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
          {error}
        </div>
      )}
    </div>
  );
}

// Page: full list of every reservation awaiting admin approval.
// One fetch, zero client-side filtering. Each row has inline
// Approve / Reject buttons. Invalidations are cross-page so the
// dashboard's pending count (KPI + preview list) stays in sync.
export function PendingReservationsPage() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin-pending-reservations'],
    queryFn: () => fetchPendingReservations(100),
  });

  function handleInvalidate() {
    queryClient.invalidateQueries({ queryKey: ['admin-pending-reservations'] });
    // Keep the dashboard's KPI + preview in sync.
    queryClient.invalidateQueries({ queryKey: ['admin-dashboard'] });
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="px-4 py-6 sm:px-6 lg:px-8">
        <div className="rounded-xl border border-destructive/40 bg-destructive/5 p-6 text-center">
          <p className="text-sm font-medium text-destructive">
            No se ha podido cargar la lista de reservas pendientes.
          </p>
        </div>
      </div>
    );
  }

  const pending = data.pending;

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight">
          <Clock className="h-6 w-6 text-amber-500" strokeWidth={2.5} />
          Reservas pendientes
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {pending.length === 0
            ? 'No hay reservas esperando aprobación.'
            : `${pending.length} reservas esperan tu aprobación. Revisa primero las más antiguas.`}
        </p>
      </div>

      {pending.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-border py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <Check className="h-8 w-8 text-emerald-500" strokeWidth={2} />
          </div>
          <p className="text-lg font-semibold">Todas las reservas están al día</p>
          <p className="text-sm text-muted-foreground">No hay pendientes de aprobación ahora mismo.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {pending.map((e) => (
            <PendingRow key={e.reservation_id} entry={e} onInvalidate={handleInvalidate} />
          ))}
        </div>
      )}
    </div>
  );
}

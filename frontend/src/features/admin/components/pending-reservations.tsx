import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Check, X, AlertCircle } from 'lucide-react';
import { confirmReservation, rejectReservation } from '@/infrastructure/repositories/reservation-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { useToast } from '@/features/ui/hooks/toast-context';
import type { ReservationView } from '@/domain/entities/reservation';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

interface PendingReservation extends ReservationView {
  owner_id: number;
  owner_name: string;
}

interface PendingReservationsProps {
  items: PendingReservation[];
  isLoading: boolean;
}

function PendingRow({ item, onInvalidate }: { item: PendingReservation; onInvalidate: () => void }) {
  const toast = useToast();
  const [error, setError] = useState('');

  const confirmMutation = useMutation({
    mutationFn: () => confirmReservation(item.owner_id, item.id),
    onSuccess: () => {
      toast.success(
        'Reserva confirmada',
        `${item.dog_name} (${item.owner_name}) queda inscrito en ${item.activity_name}.`,
      );
      onInvalidate();
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al aprobar la reserva.')),
  });

  const rejectMutation = useMutation({
    mutationFn: () => rejectReservation(item.owner_id, item.id),
    onSuccess: () => {
      toast.warning(
        'Reserva rechazada',
        `${item.dog_name} (${item.owner_name}) ha sido rechazado para ${item.activity_name}.`,
      );
      onInvalidate();
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al rechazar la reserva.')),
  });

  return (
    <div>
      <div className="flex items-center justify-between gap-3 rounded-lg border border-border p-3 transition-colors hover:bg-muted/30">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-2">
            <span className="truncate text-sm font-semibold">{item.dog_name}</span>
            <span className="text-xs text-muted-foreground">{item.owner_name}</span>
          </div>
          <div className="mt-0.5 text-xs text-muted-foreground">
            {item.activity_name} ·{' '}
            {new Date(item.activity_date).toLocaleDateString('es-ES', {
              day: 'numeric',
              month: 'short',
              hour: '2-digit',
              minute: '2-digit',
            })}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1 border-emerald-300 text-emerald-700 hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/30"
            onClick={() => { setError(''); confirmMutation.mutate(); }}
            disabled={confirmMutation.isPending || rejectMutation.isPending}
          >
            {confirmMutation.isPending ? <LoadingSpinner size="sm" /> : <Check className="h-3.5 w-3.5" />}
            <span className="hidden sm:inline">Aprobar</span>
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1 border-red-300 text-red-700 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950/30"
            onClick={() => { setError(''); rejectMutation.mutate(); }}
            disabled={confirmMutation.isPending || rejectMutation.isPending}
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

export function PendingReservations({ items, isLoading }: PendingReservationsProps) {
  const queryClient = useQueryClient();

  function handleInvalidate() {
    queryClient.invalidateQueries({ queryKey: ['admin-dashboard'], refetchType: 'all' });
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-6">
        <LoadingSpinner />
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="rounded-xl border border-border p-6 text-center">
        <p className="text-sm font-medium text-muted-foreground">No hay reservas pendientes de aprobación</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Reservas pendientes de aprobación</h3>
        <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
          {items.length}
        </span>
      </div>

      <div className="space-y-2">
        {items.map((item) => (
          <PendingRow key={item.id} item={item} onInvalidate={handleInvalidate} />
        ))}
      </div>
    </div>
  );
}

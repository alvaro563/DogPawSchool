import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { rejectReservation } from '@/infrastructure/repositories/reservation-repository.impl';
import { useToast } from '@/features/ui/hooks/toast-context';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.details) return b.details;
  return fallback;
}

interface RejectReservationButtonProps {
  reservationId: number;
  ownerId: number;
  dogName: string;
  onInvalidate: () => void;
  variant?: 'compact' | 'icon';
}

// RejectReservationButton rejects a PENDING_TO_CONFIRM reservation
// through the existing admin "reject" endpoint. Cancellation is a
// different operation (CANCELLED_IN_TIME/LATE for CONFIRMED
// reservations) — see CancelReservationButton.
//
// After success, the broader reservation query keys are
// invalidated so the admin "Reservas" page and the "Pendientes"
// dashboard both refresh.
export function RejectReservationButton({
  reservationId,
  ownerId,
  dogName,
  onInvalidate,
  variant = 'icon',
}: RejectReservationButtonProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: () => rejectReservation(ownerId, reservationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['all-reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin-pending-reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['activity-roster'] });

      toast.warning(
        'Reserva rechazada',
        `${dogName} ha sido rechazado para esta clase.`,
      );
      onInvalidate();
    },
    onError: (err: unknown) => {
      setError(parseError(err, 'Error al rechazar la reserva.'));
    },
  });

  function handleClick() {
    setError('');
    mutation.mutate();
  }

  const sizeClass = 'h-7';

  return (
    <div>
      <Button
        size="sm"
        variant="outline"
        className={`${sizeClass} gap-1 border-red-300 text-red-700 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950/30`}
        onClick={handleClick}
        disabled={mutation.isPending}
      >
        {mutation.isPending ? (
          <LoadingSpinner size="sm" />
        ) : (
          <X className="h-3.5 w-3.5" />
        )}
        {variant === 'compact' && <span className="hidden sm:inline">Rechazar</span>}
      </Button>
      {error && (
        <div className="mt-1 flex items-start gap-2 rounded-md bg-destructive/10 px-3 py-1.5 text-xs text-destructive">
          <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
          {error}
        </div>
      )}
    </div>
  );
}

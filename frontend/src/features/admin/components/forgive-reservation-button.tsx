import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, HandHeart } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { forgiveReservation } from '@/infrastructure/repositories/reservation-repository.impl';
import { useToast } from '@/features/ui/hooks/toast-context';
import { getReservationErrorMessage } from '@/features/reservations/utils/reservation-error';
import type { ApiError } from '@/infrastructure/api/http-client';

interface ForgiveReservationButtonProps {
  reservationId: number;
  dogName: string;
  onInvalidate: () => void;
  variant?: 'compact' | 'icon';
}

// ForgiveReservationButton forgives a CANCELLED_LATE reservation,
// transitioning it to FORGIVEN and refunding the pass session when
// refundable. Lives in the admin "Canceladas tarde" page next to
// the rows the action targets.
//
// No window.confirm: the action is reversible in spirit (the
// reservation stays in the DB with a different status; the admin
// can re-cancel it via the cancel endpoint if needed) and mirrors
// the immediate-fire UX of RejectReservationButton.
//
// Toast copy mirrors pass_session_refunded:
//   - true  → "Sesión perdonada y devuelta al bono"
//   - false → "Reserva perdonada (el bono no tenía nada que devolver)"
export function ForgiveReservationButton({
  reservationId,
  dogName,
  onInvalidate,
  variant = 'compact',
}: ForgiveReservationButtonProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: () => forgiveReservation(reservationId),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['all-reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin-reservations', 'cancelled-late'] });
      queryClient.invalidateQueries({ queryKey: ['reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['passes'] });

      if (data.pass_session_refunded) {
        toast.success(
          'Sesión perdonada',
          `${dogName} ha sido perdonado. La sesión se ha devuelto al bono.`,
        );
      } else {
        toast.success(
          'Reserva perdonada',
          `${dogName} ha sido perdonado (el bono no tenía nada que devolver).`,
        );
      }
      onInvalidate();
    },
    onError: (err: ApiError) => {
      setError(getReservationErrorMessage(err, 'Error al perdonar la reserva.'));
    },
  });

  function handleClick() {
    setError('');
    mutation.mutate();
  }

  return (
    <div>
      <Button
        size="sm"
        variant="outline"
        className="h-7 gap-1 border-emerald-300 text-emerald-700 hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/30"
        onClick={handleClick}
        disabled={mutation.isPending}
      >
        {mutation.isPending ? (
          <LoadingSpinner size="sm" />
        ) : (
          <HandHeart className="h-3.5 w-3.5" />
        )}
        {variant === 'compact' && <span className="hidden sm:inline">Perdonar y devolver</span>}
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

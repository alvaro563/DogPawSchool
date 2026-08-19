import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, XCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { cancelReservation, cancelReservationAdmin } from '@/infrastructure/repositories/reservation-repository.impl';
import { useToast } from '@/features/ui/hooks/toast-context';
import type { ApiError } from '@/infrastructure/api/http-client';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.details) return b.details;
  return fallback;
}

interface CancelReservationButtonProps {
  reservationId: number;
  ownerId: number;
  dogName: string;
  activityName: string;
  activityDate: string;
  isAdmin: boolean;
  onInvalidate: () => void;
  variant?: 'outline' | 'compact';
}

// CancelReservationButton renders a single button that cancels a
// CONFIRMED reservation. The check + confirm dialog are simplified
// to a window.confirm so the component stays dependency-free and
// matches the convention used by users-management.tsx for the
// deactivate button.
//
// Endpoint selection:
//   • isAdmin=true  → POST /reservations/{id}/cancel
//   • isAdmin=false → POST /users/{owner}/reservations/{id}/cancel
//
// Toast taxonomy (driven by the new status returned by the backend):
//   • CANCELLED_IN_TIME → success "Reserva cancelada" + "Sesión devuelta"
//   • CANCELLED_LATE    → warning "Reserva cancelada" + "Fuera de plazo (no se devolvió la sesión)"
//
// After success, onInvalidate is called so the calling page
// refreshes its query — typically this invalidates the broader
// 'reservations' / 'all-reservations' / 'activity-roster' keys in
// the parent's queryClient.
export function CancelReservationButton({
  reservationId,
  ownerId,
  dogName,
  activityName,
  activityDate,
  isAdmin,
  onInvalidate,
  variant = 'outline',
}: CancelReservationButtonProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: () =>
      isAdmin
        ? cancelReservationAdmin(reservationId)
        : cancelReservation(ownerId, reservationId),
    onSuccess: (data) => {
      // Wide invalidation: covers the user list, the admin
      // global list, the activity-detail roster, the pending
      // dashboard, and the dashboard KPIs. Cheap because they
      // only re-run if mounted.
      queryClient.invalidateQueries({ queryKey: ['reservations'] });
      queryClient.invalidateQueries({ queryKey: ['all-reservations'] });
      queryClient.invalidateQueries({ queryKey: ['activity-roster'] });
      queryClient.invalidateQueries({ queryKey: ['admin-pending-reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'] });

      const late = isLateCancel(activityDate);
      if (data.status === 'CANCELLED_LATE' || late) {
        toast.warning(
          'Reserva cancelada',
          'Fuera de plazo — no se devolvió la sesión.',
        );
      } else {
        toast.success(
          'Reserva cancelada',
          'Sesión devuelta al bono.',
        );
      }

      onInvalidate();
    },
    onError: (err: unknown) => {
      setError(parseError(err, 'No se pudo cancelar la reserva.'));
    },
  });

  function handleClick() {
    setError('');
    const ok = confirm(
      `¿Cancelar la reserva de ${dogName} en "${activityName}"?`,
    );
    if (!ok) return;
    mutation.mutate();
  }

  const sizeClass = variant === 'compact' ? 'h-7 px-2' : 'h-8';
  const iconClass = variant === 'compact' ? 'h-3.5 w-3.5' : 'h-4 w-4';

  return (
    <div>
      <Button
        size={variant === 'compact' ? 'sm' : 'default'}
        variant="outline"
        className={`${sizeClass} gap-1 border-red-300 text-red-700 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950/30`}
        onClick={handleClick}
        disabled={mutation.isPending}
      >
        {mutation.isPending ? (
          <LoadingSpinner size="sm" />
        ) : (
          <XCircle className={iconClass} />
        )}
        {variant === 'compact' && <span className="hidden sm:inline">Cancelar</span>}
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

function isLateCancel(activityDate: string): boolean {
  // The cancellation late window is 2h. On the client we use
  // this ONLY to pick the toast message when the backend has
  // not yet returned — the backend's status is the source of
  // truth and is preferred inside onSuccess.
  const date = new Date(activityDate);
  const now = new Date();
  return date.getTime() - now.getTime() <= 2 * 60 * 60 * 1000;
}

// Re-export the error type so consumers don't have to hunt.
export type { ApiError };

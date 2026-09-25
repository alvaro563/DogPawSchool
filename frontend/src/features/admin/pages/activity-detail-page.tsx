import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Calendar, Check, CheckCheck, Dog, Edit3, MapPin, School, User, X, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { fetchActivityRoster } from '@/infrastructure/repositories/reservation-repository.impl';
import { confirmReservation, rejectReservation } from '@/infrastructure/repositories/reservation-repository.impl';
import { bulkCompleteActivity } from '@/infrastructure/repositories/activity-repository.impl';
import { isActivityPast } from '@/features/calendar/hooks/use-calendar';
import { useToast } from '@/features/ui/hooks/toast-context';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { useAdminModal } from '@/features/admin/hooks/admin-modal-context';
import { CancelReservationButton } from '@/features/admin/components/cancel-reservation-button';
import type { ActivityRosterEntry } from '@/domain/entities/reservation';

const TYPE_LABELS: Record<string, string> = {
  SOCIALIZATION_GROUP: 'Grupo socialización',
  ROUTE: 'Ruta',
  INDIVIDUAL_CLASS: 'Individual',
  EXTRA: 'Extra',
};

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
  entry: ActivityRosterEntry;
  onInvalidate: () => void;
}

// PendingRow renders one pending attendee with inline Confirm /
// Reject buttons. Both buttons invalidate the roster query on
// success so the entry moves out of the pending section without
// a full page reload.
function PendingRow({ entry, onInvalidate }: PendingRowProps) {
  const toast = useToast();
  const [error, setError] = useState('');

  const confirmMutation = useMutation({
    mutationFn: () => confirmReservation(entry.owner_id, entry.reservation_id),
    onSuccess: () => {
      toast.success(
        'Reserva confirmada',
        `${entry.dog_name} ya tiene plaza asegurada en la clase.`,
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
        `${entry.dog_name} ha sido rechazado para esta clase.`,
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
          </div>
          <div className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground">
            <User className="h-3 w-3" />
            <span>{entry.owner_name || `Usuario #${entry.owner_id}`}</span>
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
            <span className="hidden sm:inline">Confirmar</span>
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

// ActivityDetailPage: single-activity roster view ("hoja de
// clase"). Renders the activity header plus two flat sections
// (confirmed, pending). Confirmed attendees get an inline Cancel
// button; pending attendees get inline Confirm / Reject buttons.
// All mutations invalidate the roster query so the affected row
// moves to the right section or disappears without a full page
// reload.
//
// The "Volver" button uses window.history.back() instead of a
// hardcoded route so the user lands back on whichever list they
// came from (/admin/activities or /admin/today-classes).
export function ActivityDetailPage({ id }: { id: number }) {
  // ── All hooks must be called unconditionally BEFORE any early
  // returns (Rules of Hooks). The pure derived data and JSX below
  // this comment may use early returns freely.
  const queryClient = useQueryClient();
  const toast = useToast();
  const { isAdmin } = useAuth();
  const { open: openAdminModal } = useAdminModal();

  const { data: roster, isLoading, isError } = useQuery({
    queryKey: ['activity-roster', id],
    queryFn: () => fetchActivityRoster(id),
  });

  const bulkCompleteMutation = useMutation({
    mutationFn: () => bulkCompleteActivity(id),
    onSuccess: ({ completed, closed }) => {
      // ['activity-roster'] uses the generic prefix so the roster
      // query is invalidated no matter how it was created (number
      // vs string id). Same approach is used elsewhere in this file.
      queryClient.invalidateQueries({ queryKey: ['activity-roster'] });
      // ['activities'] is the root prefix: invalidates BOTH
      // /admin/activities (open list) and /admin/activities/completed
      // in one call, so the row disappears from the open list AND
      // appears in the completed list without a second round trip.
      queryClient.invalidateQueries({ queryKey: ['activities'] });
      queryClient.invalidateQueries({ queryKey: ['reservations'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'] });
      if (completed === 0 && closed) {
        toast.success(
          'Actividad cerrada',
          'No había reservas confirmadas pendientes en esta actividad.',
        );
      } else if (completed > 0 && closed) {
        toast.success(
          'Actividad completada y cerrada',
          `${completed} ${completed === 1 ? 'reserva marcada' : 'reservas marcadas'} como COMPLETADAS.`,
        );
      }
    },
    onError: (err: unknown) => {
      const body = (err as { body?: { error?: string; details?: string } }).body;
      const code = body?.error;
      const msg =
        code === 'activity_not_finished'
          ? 'La actividad aún no ha finalizado.'
          : code === 'pending_to_confirm_exists'
          ? 'Hay reservas pendientes de confirmar. Resuélvelas primero.'
          : code === 'not_found'
          ? 'Actividad no encontrada.'
          : parseError(err, 'No se pudo completar el lote.');
      toast.error('Error al completar', msg);
    },
  });

  function handleInvalidate() {
    queryClient.invalidateQueries({ queryKey: ['activity-roster', id] });
  }

  function handleBack() {
    window.history.back();
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (isError || !roster) {
    return (
      <div className="px-4 py-6 sm:px-6 lg:px-8">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleBack}
          className="mb-4 gap-1"
        >
          <ArrowLeft className="h-4 w-4" />
          Volver
        </Button>
        <div className="rounded-xl border border-destructive/40 bg-destructive/5 p-6 text-center">
          <p className="text-sm font-medium text-destructive">
            No se ha podido cargar la hoja de clase.
          </p>
        </div>
      </div>
    );
  }

  const { activity, confirmed, pending } = roster;
  const booked = confirmed.length + pending.length;
  const pct = Math.round((booked / activity.max_capacity) * 100);

  // Gating for the bulk-complete button. The three policies:
  //   1. The activity must have finished (date + duration < now).
  //   2. There must be no PENDING_TO_CONFIRM reservation (the backend
  //      rejects the call with 409, but blocking client-side gives a
  //      better UX).
  //   3. There must be at least one CONFIRMED reservation to mark.
  // The button tooltip communicates the active reason to the admin.
  const isFinished = isActivityPast(activity.date, activity.duration_in_hours);
  const hasPending = pending.length > 0;
  const hasConfirmed = confirmed.length > 0;
  const canBulkComplete =
    isFinished && !hasPending && hasConfirmed;

  const bulkCompleteTooltip = !isFinished
    ? 'Disponible cuando la actividad haya finalizado'
    : hasPending
    ? 'Hay reservas pendientes de confirmar. Resuélvelas primero.'
    : !hasConfirmed
    ? 'Todas las reservas confirmadas ya han sido completadas'
    : `Completar ${confirmed.length} ${confirmed.length === 1 ? 'reserva' : 'reservas'}`;

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <Button
        variant="ghost"
        size="sm"
        onClick={handleBack}
        className="mb-4 gap-1"
      >
        <ArrowLeft className="h-4 w-4" />
        Volver
      </Button>

      <div className="mb-6 rounded-xl border border-border bg-card p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h1 className="text-xl font-bold tracking-tight">{activity.name}</h1>
            <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1">
                <School className="h-3 w-3" />
                {TYPE_LABELS[activity.activity_type] || activity.activity_type}
              </span>
              <span className="inline-flex items-center gap-1">
                <Calendar className="h-3 w-3" />
                {new Date(activity.date).toLocaleString('es-ES', {
                  weekday: 'short',
                  day: 'numeric',
                  month: 'short',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </span>
              <span className="inline-flex items-center gap-1">
                <MapPin className="h-3 w-3" />
                {activity.location}
              </span>
            </div>
          </div>
          <div className="shrink-0 text-right">
            <div className="text-2xl font-bold tabular-nums">{booked}</div>
            <div className="text-xs text-muted-foreground">de {activity.max_capacity} plazas</div>
            <div className="mt-1 h-1.5 w-20 overflow-hidden rounded-full bg-muted">
              <div
                className={`h-full rounded-full transition-all ${
                  pct >= 100 ? 'bg-red-500' : pct >= 70 ? 'bg-amber-500' : 'bg-sky-500'
                }`}
                style={{ width: `${Math.min(100, pct)}%` }}
              />
            </div>
            {isAdmin && !activity.closed && (
              <Button
                variant="outline"
                size="sm"
                className="mt-3 gap-1"
                onClick={() => openAdminModal('activity', activity, booked)}
              >
                <Edit3 className="h-3.5 w-3.5" />
                Editar
              </Button>
            )}
          </div>
        </div>
      </div>

      <section className="mb-6">
        <div className="mb-3 flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">Asistentes confirmados</h2>
          <div className="flex items-center gap-2">
            <span className="rounded-full bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-300">
              {confirmed.length}
            </span>
          </div>
        </div>
        {confirmed.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border p-6 text-center">
            <p className="text-sm text-muted-foreground">Aún no hay asistentes confirmados.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {confirmed.map((e) => (
              <div
                key={e.reservation_id}
                className="flex items-center justify-between gap-3 rounded-lg border border-sky-200 bg-sky-50/30 p-3 dark:border-sky-900/40 dark:bg-sky-950/10"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-x-2">
                    <Dog className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="truncate text-sm font-semibold">{e.dog_name}</span>
                  </div>
                  <div className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground">
                    <User className="h-3 w-3" />
                    <span>{e.owner_name || `Usuario #${e.owner_id}`}</span>
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <span className="rounded-full bg-sky-100 px-2 py-0.5 text-[10px] font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-300">
                    Confirmado
                  </span>
                  <CancelReservationButton
                    reservationId={e.reservation_id}
                    ownerId={e.owner_id}
                    dogName={e.dog_name}
                    activityName={activity.name}
                    activityDate={activity.date}
                    isAdmin={true}
                    onInvalidate={handleInvalidate}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold">Pendientes de confirmar</h2>
          <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
            {pending.length}
          </span>
        </div>
        {pending.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border p-6 text-center">
            <p className="text-sm text-muted-foreground">No hay perros pendientes.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {pending.map((e) => (
              <PendingRow key={e.reservation_id} entry={e} onInvalidate={handleInvalidate} />
            ))}
          </div>
        )}
      </section>

      <div className="mt-2 flex justify-center">
        <button
          type="button"
          disabled={!canBulkComplete || bulkCompleteMutation.isPending}
          title={bulkCompleteTooltip}
          onClick={() => {
            if (!window.confirm(`¿Marcar las ${confirmed.length} reservas confirmadas como COMPLETADAS y completar la actividad?`)) return;
            bulkCompleteMutation.mutate();
          }}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-3 text-sm font-semibold text-primary-foreground shadow hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {bulkCompleteMutation.isPending ? (
            <LoadingSpinner size="sm" />
          ) : (
            <CheckCheck className="h-4 w-4" />
          )}
          Completar Actividad
        </button>
      </div>
    </div>
  );
}

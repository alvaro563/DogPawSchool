import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { MapPin, Clock, Users, AlertCircle, CheckCircle2, Shield } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { fetchDogsByOwner, fetchAllActiveDogs } from '@/infrastructure/repositories/dog-repository.impl';
import { fetchPassesByUser, fetchAllPasses } from '@/infrastructure/repositories/pass-repository.impl';
import { fetchAllUsers } from '@/infrastructure/repositories/user-repository.impl';
import { createReservation, createAdminReservation } from '@/infrastructure/repositories/reservation-repository.impl';
import { formatActivityTime } from '@/features/calendar/hooks/use-calendar';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import type { Activity } from '@/domain/entities/activity';
import type { User } from '@/domain/entities/user';
import type { ApiError } from '@/infrastructure/api/http-client';

interface ActivityDetailSheetProps {
  activity: Activity | null;
  reservationStatus: string | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const TYPE_LABELS: Record<string, string> = {
  SOCIALIZATION_GROUP: 'Grupo de socialización',
  ROUTE: 'Ruta',
  INDIVIDUAL_CLASS: 'Clase individual',
  EXTRA: 'Evento extra',
};

function getErrorMessage(status: number): string {
  switch (status) {
    case 409:
      return 'No hay plazas disponibles o ya tienes una reserva.';
    case 400:
      return 'Datos inválidos. Revisa la selección.';
    case 404:
      return 'Recurso no encontrado.';
    default:
      return 'Error al crear la reserva. Inténtalo de nuevo.';
  }
}

export function ActivityDetailSheet({
  activity,
  reservationStatus,
  open,
  onOpenChange,
}: ActivityDetailSheetProps) {
  const { user, isAdmin } = useAuth();
  const queryClient = useQueryClient();
  const [selectedDogId, setSelectedDogId] = useState<number | null>(null);
  const [selectedPassId, setSelectedPassId] = useState<number | null>(null);
  const [mutationError, setMutationError] = useState('');

  const { data: dogs = [], isLoading: dogsLoading } = useQuery({
    queryKey: isAdmin ? ['dogs', 'all-active'] : ['dogs', user?.id],
    queryFn: () => (isAdmin ? fetchAllActiveDogs() : fetchDogsByOwner(user!.id)),
    enabled: open && !!user,
  });

  const { data: passes = [], isLoading: passesLoading } = useQuery({
    queryKey: isAdmin ? ['passes', 'all'] : ['passes', user?.id],
    queryFn: () => (isAdmin ? fetchAllPasses() : fetchPassesByUser(user!.id)),
    enabled: open && !!user,
  });

  const { data: users = [] } = useQuery({
    queryKey: ['users'],
    queryFn: fetchAllUsers,
    enabled: open && isAdmin,
  });

  const ownerMap = useMemo(() => {
    const map = new Map<number, User>();
    for (const u of users) map.set(u.id, u);
    return map;
  }, [users]);

  const availablePasses = useMemo(
    () => passes.filter((p) => p.remaining_sessions > 0),
    [passes],
  );

  const reservationMutation = useMutation({
    mutationFn: () => {
      const body = {
        activity_id: activity!.id,
        dog_id: selectedDogId!,
        pass_id: selectedPassId!,
      };
      return isAdmin
        ? createAdminReservation(body)
        : createReservation(user!.id, body);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['activities'] });
      queryClient.invalidateQueries({ queryKey: ['reservations'] });
      queryClient.invalidateQueries({ queryKey: ['passes'] });
      queryClient.invalidateQueries({ queryKey: ['dogs'] });
      if (isAdmin) {
        queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      }
      setMutationError('');
      setSelectedDogId(null);
      setSelectedPassId(null);
      onOpenChange(false);
    },
    onError: (err: ApiError) => {
      setMutationError(getErrorMessage(err.status));
    },
  });

  function handleReserve() {
    if (!selectedDogId || !selectedPassId) return;
    reservationMutation.mutate();
  }

  const isBooked = reservationStatus === 'CONFIRMED' || reservationStatus === 'PENDING_TO_CONFIRM';
  const canReserve = !isBooked && activity && activity.available_spots > 0 && activity.closed === false;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        {!activity ? (
          <div className="flex items-center justify-center py-20">
            <LoadingSpinner />
          </div>
        ) : (
          <>
            <SheetHeader>
              <SheetTitle className="text-xl font-bold">{activity.name}</SheetTitle>
              <div className="flex items-center gap-2">
                <p className="text-xs text-muted-foreground">
                  {TYPE_LABELS[activity.activity_type] || activity.activity_type}
                </p>
                {isAdmin && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-700 dark:bg-purple-900/30 dark:text-purple-300">
                    <Shield className="h-3 w-3" />
                    Reserva admin
                  </span>
                )}
              </div>
            </SheetHeader>

            <div className="px-4 pb-6">

            {activity.description && (
              <p className="mt-4 text-sm leading-relaxed text-muted-foreground">
                {activity.description}
              </p>
            )}

            <div className="mt-6 space-y-3">
              <div className="flex items-center gap-2 text-sm">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <span>
                  {new Date(activity.date).toLocaleDateString('es-ES', {
                    weekday: 'long',
                    day: 'numeric',
                    month: 'long',
                    year: 'numeric',
                  })}
                </span>
                <span className="font-medium">
                  {formatActivityTime(activity.date, activity.duration_in_hours)}
                </span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <MapPin className="h-4 w-4 text-muted-foreground" />
                <span>{activity.location}</span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <Users className="h-4 w-4 text-muted-foreground" />
                <span>
                  {activity.available_spots} / {activity.max_capacity} plazas disponibles
                </span>
              </div>
            </div>

            {/* Already booked message */}
            {isBooked && (
              <div className="mt-6 rounded-lg bg-sky-50 p-4 dark:bg-sky-950/30">
                <div className="flex items-center gap-2 text-sm font-medium text-sky-700 dark:text-sky-300">
                  <CheckCircle2 className="h-4 w-4" />
                  {isAdmin
                    ? (reservationStatus === 'CONFIRMED'
                      ? 'Reserva confirmada (admin)'
                      : 'Reserva pendiente de confirmación (admin)')
                    : (reservationStatus === 'CONFIRMED'
                      ? 'Ya tienes una reserva confirmada'
                      : 'Tienes una reserva pendiente de confirmación')}
                </div>
              </div>
            )}

            {/* Reservation form */}
            {canReserve && (
              <div className="mt-6 space-y-4 border-t border-border pt-6">
                <p className="text-sm font-semibold">
                  {isAdmin ? 'Reservar plaza (admin)' : 'Reservar plaza'}
                </p>

                {/* Dog selector */}
                <div className="space-y-1.5">
                  <label className="text-xs font-medium">Perro</label>
                  {dogsLoading ? (
                    <LoadingSpinner size="sm" />
                  ) : (
                    <select
                      className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
                      value={selectedDogId ?? ''}
                      onChange={(e) =>
                        setSelectedDogId(e.target.value ? Number(e.target.value) : null)
                      }
                    >
                      <option value="">Selecciona un perro</option>
                      {dogs.map((d) => (
                        <option key={d.id} value={d.id}>
                          {isAdmin
                            ? `${d.name} · ${ownerMap.get(d.user_id)?.name ?? 'Propietario #' + d.user_id}`
                            : d.name}
                        </option>
                      ))}
                    </select>
                  )}
                </div>

                {/* Pass selector */}
                <div className="space-y-1.5">
                  <label className="text-xs font-medium">Bono</label>
                  {passesLoading ? (
                    <LoadingSpinner size="sm" />
                  ) : (
                    <select
                      className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
                      value={selectedPassId ?? ''}
                      onChange={(e) =>
                        setSelectedPassId(e.target.value ? Number(e.target.value) : null)
                      }
                    >
                      <option value="">Selecciona un bono</option>
                      {availablePasses.map((p) => (
                        <option key={p.id} value={p.id}>
                          {isAdmin
                            ? `Bono ${p.pass_type === 'GENERICO' ? 'genérico' : 'específico'} · ${ownerMap.get(p.user_id)?.name ?? 'Usuario #' + p.user_id} · ${p.remaining_sessions} sesiones`
                            : `Bono ${p.pass_type === 'GENERICO' ? 'genérico' : 'específico'} — ${p.remaining_sessions} sesiones`}
                        </option>
                      ))}
                    </select>
                  )}
                </div>

                {mutationError && (
                  <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                    <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                    <span>{mutationError}</span>
                  </div>
                )}

                <Button
                  className="w-full"
                  disabled={!selectedDogId || !selectedPassId || reservationMutation.isPending}
                  onClick={handleReserve}
                >
                  {reservationMutation.isPending ? (
                    <LoadingSpinner size="sm" className="border-t-background" />
                  ) : (
                    'Confirmar reserva'
                  )}
                </Button>
              </div>
            )}

            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

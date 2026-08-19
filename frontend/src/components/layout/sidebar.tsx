import { useMemo, useState } from 'react';
import { Link, useLocation } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarDays, Dog, Ticket, ClipboardList, User, LayoutDashboard, Users, School, CreditCard, PlusCircle, Shield, ChevronDown, AlertCircle } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { useAdminModal } from '@/features/admin/hooks/admin-modal-context';
import { useSelectedActivity } from '@/features/calendar/hooks/selected-activity-context';
import { useToast } from '@/features/ui/hooks/toast-context';
import { fetchDogsByOwner } from '@/infrastructure/repositories/dog-repository.impl';
import { fetchPassesByUser } from '@/infrastructure/repositories/pass-repository.impl';
import { fetchUserReservations } from '@/infrastructure/repositories/reservation-repository.impl';
import { createReservation } from '@/infrastructure/repositories/reservation-repository.impl';
import type { CreateReservationResponse } from '@/domain/entities/reservation';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface NavItem {
  to: string;
  label: string;
  icon: React.ComponentType<{ className?: string; strokeWidth?: number }>;
}

const userNavItems: NavItem[] = [
  { to: '/calendar', label: 'Calendario', icon: CalendarDays },
  { to: '/dogs', label: 'Mis Perros', icon: Dog },
  { to: '/passes', label: 'Mis Bonos', icon: Ticket },
  { to: '/reservations', label: 'Mis Reservas', icon: ClipboardList },
  { to: '/profile', label: 'Perfil', icon: User },
];

const adminNavItems: NavItem[] = [
  { to: '/admin', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/calendar', label: 'Calendario', icon: CalendarDays },
  { to: '/admin/users', label: 'Usuarios', icon: Users },
  { to: '/admin/dogs', label: 'Perros', icon: Dog },
  { to: '/admin/activities', label: 'Actividades', icon: School },
  { to: '/admin/passes', label: 'Bonos', icon: CreditCard },
  { to: '/admin/reservations', label: 'Reservas', icon: ClipboardList },
];

interface SidebarProps {
  className?: string;
  onNavigate?: () => void;
}

const OCCUPYING_STATUSES = new Set(['CONFIRMED', 'PENDING_TO_CONFIRM']);

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

// ReserveOtherDogForm is the inline form rendered in the sidebar
// under "Reservar otro perro en esta actividad". It reuses the
// regular reservation endpoint: the backend already accepts any
// number of (activity, dog) pairs from the same user. The form
// only renders dogs the user owns AND has not yet booked for this
// activity, so the user can never accidentally double-book.
function ReserveOtherDogForm({
  activityId,
  onSuccess,
}: {
  activityId: number;
  onSuccess: () => void;
}) {
  const { user } = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [selectedDogId, setSelectedDogId] = useState<number | null>(null);
  const [selectedPassId, setSelectedPassId] = useState<number | null>(null);
  const [error, setError] = useState('');

  const { data: dogs = [] } = useQuery({
    queryKey: ['dogs', user?.id],
    queryFn: () => fetchDogsByOwner(user!.id),
    enabled: !!user,
  });
  const { data: reservations = [] } = useQuery({
    queryKey: ['reservations', user?.id],
    queryFn: () => fetchUserReservations(user!.id),
    enabled: !!user,
  });
  const { data: passes = [] } = useQuery({
    queryKey: ['passes', user?.id],
    queryFn: () => fetchPassesByUser(user!.id),
    enabled: !!user,
  });

  const reservedDogIds = useMemo(() => {
    const ids = new Set<number>();
    for (const r of reservations) {
      if (r.activity_id === activityId && OCCUPYING_STATUSES.has(r.status)) {
        ids.add(r.dog_id);
      }
    }
    return ids;
  }, [reservations, activityId]);

  const availableDogs = useMemo(
    () => dogs.filter((d) => !reservedDogIds.has(d.id)),
    [dogs, reservedDogIds],
  );
  const availablePasses = useMemo(
    () => passes.filter((p) => p.remaining_sessions > 0),
    [passes],
  );

  const mutation = useMutation({
    mutationFn: () =>
      createReservation(user!.id, {
        activity_id: activityId,
        dog_id: selectedDogId!,
        pass_id: selectedPassId!,
      }),
    onSuccess: (data: CreateReservationResponse) => {
      queryClient.invalidateQueries({ queryKey: ['reservations', user!.id] });
      queryClient.invalidateQueries({ queryKey: ['dogs', user!.id] });
      queryClient.invalidateQueries({ queryKey: ['passes', user!.id] });
      queryClient.invalidateQueries({ queryKey: ['activities'] });

      const selectedDog = dogs.find((d) => d.id === selectedDogId);
      const dogName = selectedDog?.name ?? 'Tu perro';
      if (data.status === 'PENDING_TO_CONFIRM') {
        toast.warning(
          'Reserva pendiente de confirmación',
          `${dogName} queda en lista de espera por incompatibilidades. La escuela revisará la reserva y te avisaremos.`,
        );
      } else {
        toast.success(
          '¡Reserva confirmada!',
          `${dogName} ya tiene plaza asegurada.`,
        );
      }

      onSuccess();
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al crear la reserva.')),
  });

  return (
    <div className="space-y-2 rounded-lg border border-amber-200 bg-amber-50/40 p-2.5 dark:border-amber-900/40 dark:bg-amber-950/10">
      <div className="space-y-1">
        <label className="text-[10px] font-medium text-muted-foreground">Perro</label>
        {availableDogs.length === 0 ? (
          <p className="text-xs text-muted-foreground">Todos tus perros están ya reservados.</p>
        ) : (
          <select
            className="w-full rounded border border-input bg-transparent px-2 py-1.5 text-xs"
            value={selectedDogId ?? ''}
            onChange={(e) => setSelectedDogId(e.target.value ? Number(e.target.value) : null)}
          >
            <option value="">Selecciona un perro</option>
            {availableDogs.map((d) => (
              <option key={d.id} value={d.id}>{d.name}</option>
            ))}
          </select>
        )}
      </div>
      <div className="space-y-1">
        <label className="text-[10px] font-medium text-muted-foreground">Bono</label>
        <select
          className="w-full rounded border border-input bg-transparent px-2 py-1.5 text-xs"
          value={selectedPassId ?? ''}
          onChange={(e) => setSelectedPassId(e.target.value ? Number(e.target.value) : null)}
        >
          <option value="">Selecciona un bono</option>
          {availablePasses.map((p) => (
            <option key={p.id} value={p.id}>
              Bono {p.pass_type === 'GENERICO' ? 'genérico' : 'específico'} — {p.remaining_sessions} sesiones
            </option>
          ))}
        </select>
      </div>
      {error && (
        <div className="flex items-start gap-1 rounded bg-destructive/10 px-2 py-1 text-[10px] text-destructive">
          <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
          {error}
        </div>
      )}
      <Button
        size="sm"
        className="h-7 w-full text-xs"
        disabled={!selectedDogId || !selectedPassId || mutation.isPending}
        onClick={() => {
          setError('');
          mutation.mutate();
        }}
      >
        {mutation.isPending ? (
          <LoadingSpinner size="sm" className="border-t-background" />
        ) : (
          'Confirmar reserva'
        )}
      </Button>
    </div>
  );
}

export function Sidebar({ className, onNavigate }: SidebarProps) {
  const { user, isAdmin } = useAuth();
  const location = useLocation();
  const { open } = useAdminModal();
  const { activity: selectedActivity } = useSelectedActivity();

  const navItems = isAdmin ? adminNavItems : userNavItems;

  // The "Reservar otro perro" quick action is only meaningful for
  // regular users. It requires the sidebar to know whether the
  // user has multiple dogs AND has at least one slot-holding
  // reservation for the selected activity AND has at least one
  // dog not yet booked for it. We load these on demand so the
  // sidebar stays cheap when no activity is selected.
  const showReserveOtherDogEntry = !isAdmin && !!user && !!selectedActivity;

  const { data: dogs = [], isLoading: dogsLoading } = useQuery({
    queryKey: ['dogs', user?.id],
    queryFn: () => fetchDogsByOwner(user!.id),
    enabled: !!user,
  });
  const { data: reservations = [] } = useQuery({
    queryKey: ['reservations', user?.id],
    queryFn: () => fetchUserReservations(user!.id),
    enabled: !!user,
  });

  const reservedDogIds = useMemo(() => {
    if (!selectedActivity) return new Set<number>();
    const ids = new Set<number>();
    for (const r of reservations) {
      if (r.activity_id === selectedActivity.id && OCCUPYING_STATUSES.has(r.status)) {
        ids.add(r.dog_id);
      }
    }
    return ids;
  }, [reservations, selectedActivity]);

  const userHasMultipleDogs = dogs.length >= 2;
  const userHasAnyReservationForActivity = useMemo(() => {
    if (!selectedActivity) return false;
    return reservations.some(
      (r) => r.activity_id === selectedActivity.id && OCCUPYING_STATUSES.has(r.status),
    );
  }, [reservations, selectedActivity]);
  const hasBookableDog = useMemo(() => {
    if (!selectedActivity) return false;
    return dogs.some((d) => !reservedDogIds.has(d.id));
  }, [dogs, reservedDogIds, selectedActivity]);

  const canShowReserveOtherDogButton =
    showReserveOtherDogEntry &&
    !dogsLoading &&
    userHasMultipleDogs &&
    userHasAnyReservationForActivity &&
    hasBookableDog &&
    selectedActivity.closed === false &&
    selectedActivity.available_spots > 0;

  const [reserveFormOpen, setReserveFormOpen] = useState(false);

  function handleQuickAction(action: 'activity' | 'pass' | 'dog' | 'client') {
    open(action);
    onNavigate?.();
  }

  return (
    <nav className={cn('flex flex-col gap-1 px-3 py-4', className)}>
      {user && !isAdmin && (
        <div className="mb-4 rounded-lg bg-muted/50 px-3 py-3">
          <p className="text-xs font-medium text-muted-foreground">Hola,</p>
          <p className="truncate text-sm font-semibold">{user.name}</p>
        </div>
      )}

      {navItems.map((item) => {
        const Icon = item.icon;
        const isActive = item.to === '/admin'
          ? location.pathname === '/admin' || location.pathname === '/admin/'
          : location.pathname.startsWith(item.to);
        return (
          <Link
            key={item.to}
            to={item.to as '/'}
            onClick={onNavigate}
            className={cn(
              'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
              isActive
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
          >
            <Icon className="h-4 w-4" strokeWidth={2} />
            {item.label}
          </Link>
        );
      })}

      {/* Quick action for regular users: book another dog for the
          currently selected calendar activity. Visible only when
          the user has 2+ dogs AND already holds a slot AND has a
          dog not yet booked AND the activity has free spots. */}
      {canShowReserveOtherDogButton && selectedActivity && (
        <>
          <Separator text="Acciones" />
          <button
            onClick={() => setReserveFormOpen((v) => !v)}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <Dog className="h-4 w-4" strokeWidth={2} />
            <span className="flex-1 text-left">Reservar otro perro en esta actividad</span>
            <ChevronDown
              className={cn('h-4 w-4 transition-transform', reserveFormOpen && 'rotate-180')}
              strokeWidth={2}
            />
          </button>
          {reserveFormOpen && (
            <div className="px-1">
              <ReserveOtherDogForm
                activityId={selectedActivity.id}
                onSuccess={() => {
                  setReserveFormOpen(false);
                  onNavigate?.();
                }}
              />
            </div>
          )}
        </>
      )}

      {isAdmin && (
        <>
          <Separator text="Gestión de escuela" />

          <button
            onClick={() => handleQuickAction('activity')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <PlusCircle className="h-4 w-4" strokeWidth={2} />
            Crear Actividad
          </button>

          <button
            onClick={() => handleQuickAction('pass')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <Ticket className="h-4 w-4" strokeWidth={2} />
            Registrar Bono
          </button>

          <button
            onClick={() => handleQuickAction('dog')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <Dog className="h-4 w-4" strokeWidth={2} />
            Registrar Perro
          </button>

          <button
            onClick={() => handleQuickAction('client')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <User className="h-4 w-4" strokeWidth={2} />
            Alta Cliente
          </button>

          <Link
            to={'/incompatibilities' as '/'}
            onClick={onNavigate}
            className={cn(
              'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
              location.pathname.startsWith('/incompatibilities')
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
          >
            <Shield className="h-4 w-4" strokeWidth={2} />
            Crear Rasgo / Incompatibilidad
          </Link>
        </>
      )}
    </nav>
  );
}

function Separator({ text }: { text: string }) {
  return (
    <div className="flex items-center gap-2 px-3 py-1">
      <div className="h-px flex-1 bg-border" />
      {text && <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{text}</span>}
      <div className="h-px flex-1 bg-border" />
    </div>
  );
}

import { Link, useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Calendar, MapPin, Users, School, ChevronRight, ArrowRight, Edit3 } from 'lucide-react';
import { fetchActivities } from '@/infrastructure/repositories/activity-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { useAdminModal } from '@/features/admin/hooks/admin-modal-context';

const TYPE_LABELS: Record<string, string> = {
  SOCIALIZATION_GROUP: 'Grupo socialización',
  ROUTE: 'Ruta',
  INDIVIDUAL_CLASS: 'Individual',
  EXTRA: 'Extra',
};

// ActivitiesManagementPage: open activities only. Closed ones have
// moved to /admin/activities/completed which follows the perros
// (active/inactive) precedent. The "back" button on the detail page
// uses history.back() so the user lands back here regardless of
// which list they came from.
export function ActivitiesManagementPage() {
  const navigate = useNavigate();
  const { isAdmin } = useAuth();
  const { open: openAdminModal } = useAdminModal();

  // Hierarchical query key: 'activities' is the root prefix; the
  // mutation invalidates that root to refresh both this list and
  // the completed one.
  const { data: activities = [], isLoading } = useQuery({
    queryKey: ['activities', 'list', { closed: false }],
    queryFn: () => fetchActivities('2000-01-01T00:00:00Z', '2100-01-01T00:00:00Z', false),
  });

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Actividades</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {activities.length} {activities.length === 1 ? 'actividad abierta' : 'actividades abiertas'} · pulsa una para ver la hoja de clase
          </p>
        </div>
        <Link
          to="/admin/activities/completed"
          className="inline-flex items-center gap-1 self-start rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted sm:self-auto"
        >
          Ver actividades completadas
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>

      {activities.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <School className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">No hay actividades abiertas</p>
          <p className="text-sm text-muted-foreground">
            Las actividades completadas están en{' '}
            <Link to="/admin/activities/completed" className="text-foreground underline">
              este listado
            </Link>
            .
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {activities.map((a) => {
            const booked = a.max_capacity - a.available_spots;
            const pct = Math.round((booked / a.max_capacity) * 100);

            return (
              // Outer container (not a button): the row navigates via
              // the inner button and the edit icon sits beside it —
              // a <button> inside a <button> would be invalid HTML.
              <div
                key={a.id}
                className="flex items-center gap-1 rounded-xl border border-border bg-card pr-2 transition-colors hover:bg-muted/30"
              >
                <button
                  type="button"
                  onClick={() =>
                    navigate({
                      to: '/admin/activities/$id',
                      params: { id: String(a.id) },
                    })
                  }
                  className="flex min-w-0 flex-1 items-center gap-3 p-4 text-left"
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold">{a.name}</p>
                    <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                      <span className="inline-flex items-center gap-1">
                        <School className="h-3 w-3" />
                        {TYPE_LABELS[a.activity_type] || a.activity_type}
                      </span>
                      <span>·</span>
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {new Date(a.date).toLocaleDateString('es-ES', {
                          day: 'numeric',
                          month: 'short',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                      <span>·</span>
                      <span className="flex items-center gap-1">
                        <MapPin className="h-3 w-3" />
                        {a.location}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-3 shrink-0">
                    <div className="flex items-center gap-1.5">
                      <div className="h-2 w-16 overflow-hidden rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full ${
                            pct >= 100 ? 'bg-red-500' : pct >= 70 ? 'bg-amber-500' : 'bg-sky-500'
                          }`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                      <span className="text-xs tabular-nums text-muted-foreground">{booked}/{a.max_capacity}</span>
                    </div>
                    <Users className="h-4 w-4 text-muted-foreground" />
                    <ChevronRight className="h-4 w-4 text-muted-foreground" />
                  </div>
                </button>
                {isAdmin && (
                  <button
                    type="button"
                    title="Editar"
                    aria-label={`Editar ${a.name}`}
                    onClick={() => openAdminModal('activity', a, booked)}
                    className="shrink-0 rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground"
                  >
                    <Edit3 className="h-4 w-4" />
                  </button>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

import { Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Calendar, MapPin, Users, School, ChevronRight, ArrowLeft, CheckCheck } from 'lucide-react';
import { fetchActivities } from '@/infrastructure/repositories/activity-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';

const TYPE_LABELS: Record<string, string> = {
  SOCIALIZATION_GROUP: 'Grupo socialización',
  ROUTE: 'Ruta',
  INDIVIDUAL_CLASS: 'Individual',
  EXTRA: 'Extra',
};

// CompletedActivitiesPage: mirror of /admin/activities limited to
// closed activities. Follows the perros (active/inactive) precedent:
// two separate pages, both within the same URL prefix. Navigation
// uses a declarative <Link to="/admin/activities"> instead of
// history.back() so F5 / direct URL entry work correctly.
export function CompletedActivitiesPage() {
  // Hierarchical query key, mirroring the open-list page; the
  // mutation invalidates the 'activities' root prefix to refresh
  // both lists together.
  const { data: activities = [], isLoading } = useQuery({
    queryKey: ['activities', 'list', { closed: true }],
    queryFn: () => fetchActivities('2000-01-01T00:00:00Z', '2100-01-01T00:00:00Z', true),
  });

  if (isLoading) {
    return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;
  }

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <Link
        to="/admin/activities"
        className="mb-4 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Volver a actividades
      </Link>

      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Actividades completadas</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {activities.length} {activities.length === 1 ? 'actividad completada' : 'actividades completadas'} · pulsa una para revisar la hoja de clase
        </p>
      </div>

      {activities.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border bg-card py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <CheckCheck className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">No hay actividades completadas todavía</p>
          <p className="text-sm text-muted-foreground">
            Cuando cierres una actividad (manual o desde "Completar Actividad") aparecerá aquí.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {activities.map((a) => {
            const booked = a.max_capacity - a.available_spots;

            return (
              <Link
                key={a.id}
                to="/admin/activities/$id"
                params={{ id: String(a.id) }}
                className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-muted/30"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="truncate text-sm font-semibold">{a.name}</p>
                    <span className="rounded bg-emerald-100 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                      Completada
                    </span>
                  </div>
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
                    <div className="h-2 w-16 overflow-hidden rounded-full bg-emerald-500" />
                    <span className="text-xs tabular-nums text-muted-foreground">{booked}/{a.max_capacity}</span>
                  </div>
                  <Users className="h-4 w-4 text-muted-foreground" />
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

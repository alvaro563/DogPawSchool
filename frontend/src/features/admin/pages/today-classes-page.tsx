import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { Calendar, MapPin, Users, School, ChevronRight } from 'lucide-react';
import { fetchActivities } from '@/infrastructure/repositories/activity-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';

const TYPE_LABELS: Record<string, string> = {
  SOCIALIZATION_GROUP: 'Grupo socialización',
  ROUTE: 'Ruta',
  INDIVIDUAL_CLASS: 'Individual',
  EXTRA: 'Extra',
};

// Page: full list of every class scheduled for the current day.
// Anchored to the user's local midnight so "today" matches the
// calendar and the dashboard. Used as the entry point from the
// "Clases hoy" KPI card on the admin dashboard.
export function TodayClassesPage() {
  const navigate = useNavigate();

  const { from, to } = useMemo(() => {
    const start = new Date();
    start.setHours(0, 0, 0, 0);
    const end = new Date(start);
    end.setDate(end.getDate() + 1);
    return {
      from: start.toISOString().replace(/\.\d{3}/, ''),
      to: end.toISOString().replace(/\.\d{3}/, ''),
    };
  }, []);

  const { data: activities = [], isLoading } = useQuery({
    queryKey: ['today-classes', from, to],
    queryFn: () => fetchActivities(from, to),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Clases de hoy</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Pulsa una clase para ver los asistentes y los perros pendientes de confirmar.
        </p>
      </div>

      {activities.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-border py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <School className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">No hay clases programadas para hoy</p>
        </div>
      ) : (
        <div className="space-y-2">
          {activities.map((a) => {
            const booked = a.max_capacity - a.available_spots;
            const pct = Math.round((booked / a.max_capacity) * 100);

            return (
              <button
                key={a.id}
                onClick={() =>
                  navigate({
                    to: '/admin/today-classes/$id',
                    params: { id: String(a.id) },
                  })
                }
                className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-muted/30"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="truncate text-sm font-semibold">{a.name}</p>
                    {a.closed && (
                      <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                        Cerrada
                      </span>
                    )}
                  </div>
                  <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                    <span className="inline-flex items-center gap-1">
                      <School className="h-3 w-3" />
                      {TYPE_LABELS[a.activity_type] || a.activity_type}
                    </span>
                    <span>·</span>
                    <span className="flex items-center gap-1">
                      <Calendar className="h-3 w-3" />
                      {new Date(a.date).toLocaleTimeString('es-ES', {
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
                    <div className="h-2 w-20 overflow-hidden rounded-full bg-muted">
                      <div
                        className={`h-full rounded-full transition-all ${
                          pct >= 100
                            ? 'bg-red-500'
                            : pct >= 70
                            ? 'bg-amber-500'
                            : 'bg-sky-500'
                        }`}
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                    <span className="text-xs font-medium tabular-nums text-muted-foreground">
                      {booked}/{a.max_capacity}
                    </span>
                  </div>
                  <Users className="h-4 w-4 text-muted-foreground" />
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

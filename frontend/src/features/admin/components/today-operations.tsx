import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Users } from 'lucide-react';
import { formatActivityTime } from '@/features/calendar/hooks/use-calendar';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import apiClient from '@/infrastructure/api/http-client';
import type { Activity } from '@/domain/entities/activity';
import type { ReservationView } from '@/domain/entities/reservation';

interface TodayOperationsProps {
  activities: Activity[];
  isLoading: boolean;
}

function AttendeesSheet({
  activity,
  open,
  onOpenChange,
}: {
  activity: Activity;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { data: reservations = [], isLoading } = useQuery({
    queryKey: ['activity-reservations', activity.id],
    queryFn: async () => {
      const data = await apiClient.get<{ reservations: ReservationView[] }>(
        `/activities/${activity.id}/reservations`,
        { limit: '100' },
      );
      return data.reservations.filter(
        (r) => r.status === 'CONFIRMED' || r.status === 'PENDING_TO_CONFIRM',
      );
    },
    enabled: open,
  });

  const booked = activity.max_capacity - activity.available_spots;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="text-lg font-bold">{activity.name}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            {booked} / {activity.max_capacity} plazas ocupadas
          </p>
        </SheetHeader>

        {isLoading ? (
          <div className="flex justify-center py-10">
            <LoadingSpinner />
          </div>
        ) : reservations.length === 0 ? (
          <p className="py-10 text-center text-sm text-muted-foreground">Sin asistentes aún</p>
        ) : (
          <div className="mt-4 space-y-2 px-4">
            {reservations.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-lg border border-border p-3">
                <div>
                  <p className="text-sm font-semibold">{r.dog_name}</p>
                </div>
                <span
                  className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                    r.status === 'CONFIRMED'
                      ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
                      : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                  }`}
                >
                  {r.status === 'CONFIRMED' ? 'Confirmado' : 'Pendiente'}
                </span>
              </div>
            ))}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}

export function TodayOperations({ activities, isLoading }: TodayOperationsProps) {
  const [selectedActivity, setSelectedActivity] = useState<Activity | null>(null);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-6">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <>
      <div>
        <h3 className="mb-3 text-sm font-semibold">Operativa de hoy</h3>
        {activities.length === 0 ? (
          <div className="rounded-xl border border-border p-6 text-center">
            <p className="text-sm text-muted-foreground">No hay clases programadas para hoy</p>
          </div>
        ) : (
          <div className="space-y-2">
            {activities.map((activity) => {
              const booked = activity.max_capacity - activity.available_spots;
              const pct = Math.round((booked / activity.max_capacity) * 100);

              return (
                <button
                  key={activity.id}
                  onClick={() => setSelectedActivity(activity)}
                  className="flex w-full items-center gap-3 rounded-lg border border-border p-3 text-left transition-colors hover:bg-muted/30"
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold">{activity.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {formatActivityTime(activity.date, activity.duration_in_hours)}
                    </p>
                  </div>

                  <div className="flex shrink-0 items-center gap-2">
                    <div className="flex items-center gap-1.5">
                      <div className="h-2 w-20 overflow-hidden rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full transition-all ${
                            pct >= 100 ? 'bg-red-500' : pct >= 70 ? 'bg-amber-500' : 'bg-sky-500'
                          }`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                      <span className="text-xs font-medium tabular-nums text-muted-foreground">
                        {booked}/{activity.max_capacity}
                      </span>
                    </div>
                    <Users className="h-4 w-4 text-muted-foreground" />
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {selectedActivity && (
        <AttendeesSheet
          activity={selectedActivity}
          open={!!selectedActivity}
          onOpenChange={(open) => {
            if (!open) setSelectedActivity(null);
          }}
        />
      )}
    </>
  );
}

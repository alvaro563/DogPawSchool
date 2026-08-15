import { useMemo } from 'react';
import { cn } from '@/lib/utils';
import type { Activity } from '@/domain/entities/activity';
import { ActivityCard } from './activity-card';
import { layoutOverlappingActivities } from '@/features/calendar/utils/calendar-layout';

interface DayViewProps {
  currentDate: Date;
  activities: Activity[];
  userReservationMap: Map<number, string>;
  onActivityClick: (activity: Activity) => void;
}

const HOURS = Array.from({ length: 13 }, (_, i) => i + 8);
const HOUR_HEIGHT = 64;

export function DayView({
  currentDate,
  activities,
  userReservationMap,
  onActivityClick,
}: DayViewProps) {
  const todayStr = new Date().toDateString();
  const isToday = currentDate.toDateString() === todayStr;

  const dayActivities = useMemo(() => {
    const dayStr = currentDate.toDateString();
    return activities.filter((a) => new Date(a.date).toDateString() === dayStr);
  }, [activities, currentDate]);

  const positionedActivities = useMemo(
    () => layoutOverlappingActivities(dayActivities),
    [dayActivities],
  );

  return (
    <div className="flex flex-1 flex-col">
      {/* Day header */}
      <div className="border-b border-border px-4 py-3 text-center">
        <p
          className={cn(
            'text-sm font-semibold',
            isToday ? 'text-primary' : 'text-foreground',
          )}
        >
          {currentDate.toLocaleDateString('es-ES', {
            weekday: 'long',
            day: 'numeric',
            month: 'long',
          })}
        </p>
      </div>

      {/* Time grid */}
      <div className="relative flex-1 overflow-auto">
        <div className="relative min-w-[520px]" style={{ height: HOURS.length * HOUR_HEIGHT }}>
        {HOURS.map((hour) => (
          <div
            key={hour}
            className="flex border-b border-border"
            style={{ height: HOUR_HEIGHT }}
          >
            <div className="w-14 shrink-0 border-r border-border pr-1 pt-0 text-right text-[10px] text-muted-foreground">
              {String(hour).padStart(2, '0')}:00
            </div>
            <div className="flex-1" />
          </div>
        ))}

        <div className="absolute inset-y-0 right-0 left-14">
          {positionedActivities.map(({ activity, startMinutes, endMinutes, column, totalColumns }) => {
            const top = (startMinutes - 8 * 60) * (HOUR_HEIGHT / 60);
            if (top + (endMinutes - startMinutes) * (HOUR_HEIGHT / 60) < 0) return null;
            return (
              <div
                key={activity.id}
                className="absolute"
                style={{
                  top: Math.max(0, top),
                  height: Math.max((endMinutes - startMinutes) * (HOUR_HEIGHT / 60), 48),
                  left: `${(column * 100) / totalColumns}%`,
                  width: `calc(${100 / totalColumns}% - 2px)`,
                }}
              >
                <ActivityCard
                  activity={activity}
                  reservationStatus={userReservationMap.get(activity.id)}
                  onClick={onActivityClick}
                />
              </div>
            );
          })}
        </div>
        </div>
      </div>
    </div>
  );
}

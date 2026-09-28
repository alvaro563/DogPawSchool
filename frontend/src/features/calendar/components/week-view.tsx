import { useMemo } from 'react';
import { cn } from '@/lib/utils';
import type { Activity } from '@/domain/entities/activity';
import { ActivityCard } from './activity-card';
import { layoutOverlappingActivities } from '@/features/calendar/utils/calendar-layout';

interface WeekViewProps {
  currentDate: Date;
  activities: Activity[];
  userReservationMap: Map<number, string>;
  onActivityClick: (activity: Activity) => void;
}

const HOURS = Array.from({ length: 13 }, (_, i) => i + 8);
const HOUR_HEIGHT = 60;

function getWeekDays(date: Date): Date[] {
  const day = date.getDay();
  const monday = new Date(date);
  monday.setDate(date.getDate() - day + (day === 0 ? -6 : 1));
  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    return d;
  });
}

type DayActivityMap = Map<string, Activity[]>;

export function WeekView({
  currentDate,
  activities,
  userReservationMap,
  onActivityClick,
}: WeekViewProps) {
  const weekDays = useMemo(() => getWeekDays(currentDate), [currentDate]);
  const todayStr = new Date().toDateString();

  const dayActivityMap: DayActivityMap = useMemo(() => {
    const map = new Map<string, Activity[]>();
    for (const a of activities) {
      const dayKey = new Date(a.date).toDateString();
      const list = map.get(dayKey) || [];
      list.push(a);
      map.set(dayKey, list);
    }
    return map;
  }, [activities]);

  return (
    <div className="flex flex-1 flex-col">
      {/* Day headers + time grid share ONE scroll container: both
          grids must scroll horizontally together and use the same
          intrinsic width (min-w-[720px]). Previously the header sat
          OUTSIDE the overflow-auto container with no min-w, so on
          mobile it shrank to the viewport (~49px columns) while the
          cells forced 720px (~96px columns) and the columns no longer
          matched the day titles. On desktop the container is already
          ≥720px, which is why it only reproduced on mobile. */}
      <div className="relative flex-1 overflow-auto">
        {/* Day headers: sticky so vertical scrolling keeps them in
            view (they used to live outside the scroll container and
            were pinned as a side effect). Opaque background so the
            activity blocks pass underneath without showing through. */}
        <div className="sticky top-0 z-10 grid min-w-[720px] grid-cols-[3rem_repeat(7,1fr)] border-b border-border bg-background">
          <div className="border-r border-border" />
          {weekDays.map((day) => (
            <div
              key={day.toISOString()}
              className={cn(
                'border-r border-border px-1 py-2 text-center text-xs font-medium',
                day.toDateString() === todayStr
                  ? 'text-primary'
                  : 'text-muted-foreground',
              )}
            >
              <div className="hidden sm:block">
                {day.toLocaleDateString('es-ES', { weekday: 'short' })}
              </div>
              <div className="sm:hidden">
                {day.toLocaleDateString('es-ES', { weekday: 'narrow' })}
              </div>
              <div className="text-[10px]">{day.getDate()}</div>
            </div>
          ))}
        </div>

        {/* Time grid */}
        <div
          className="relative grid min-w-[720px] grid-cols-[3rem_repeat(7,1fr)]"
          style={{ height: HOURS.length * HOUR_HEIGHT }}
        >
          {HOURS.map((hour) => (
            <div
              key={hour}
              className="contents"
              style={{ height: HOUR_HEIGHT }}
            >
              <div className="border-r border-border pr-1 pt-0 text-right text-[10px] text-muted-foreground">
                {String(hour).padStart(2, '0')}:00
              </div>
              {weekDays.map((day) => (
                <div
                  key={day.toISOString()}
                  className="border-b border-r border-border"
                  style={{ height: HOUR_HEIGHT }}
                />
              ))}
            </div>
          ))}

          {/* Activities positioned absolutely */}
          {weekDays.map((day, dayIndex) => {
            const dayStr = day.toDateString();
            const dayActs = dayActivityMap.get(dayStr) || [];
            const positioned = layoutOverlappingActivities(dayActs);
            if (positioned.length === 0) return null;

            return (
              <div
                key={dayStr}
                className="absolute top-0 bottom-0"
                style={{
                  left: `calc(${(dayIndex * 100) / 7}% + ${3 * (1 - dayIndex / 7)}rem)`,
                  width: `calc(${100 / 7}% - ${3 / 7}rem)`,
                }}
              >
                {positioned.map(({ activity, startMinutes, endMinutes, column, totalColumns }) => {
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
                        compact
                        density="comfortable"
                      />
                    </div>
                  );
                })}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

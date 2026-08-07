import { useMemo } from 'react';
import { cn } from '@/lib/utils';
import type { Activity } from '@/domain/entities/activity';
import { ActivityCard } from './activity-card';

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

  function getTop(activity: Activity): number {
    const d = new Date(activity.date);
    const hours = d.getHours();
    const minutes = d.getMinutes();
    return (hours - 8) * HOUR_HEIGHT + (minutes / 60) * HOUR_HEIGHT;
  }

  function getHeight(activity: Activity): number {
    return activity.duration_in_hours * HOUR_HEIGHT;
  }

  return (
    <div className="flex flex-1 flex-col">
      {/* Day headers */}
      <div className="grid grid-cols-[3rem_repeat(7,1fr)] border-b border-border">
        <div />
        {weekDays.map((day) => (
          <div
            key={day.toISOString()}
            className={cn(
              'px-1 py-2 text-center text-xs font-medium',
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
      <div className="relative flex-1 overflow-auto">
        <div className="grid grid-cols-[3rem_repeat(7,1fr)]">
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
          {weekDays.map((day) => {
            const dayStr = day.toDateString();
            const dayActs = dayActivityMap.get(dayStr) || [];
            return dayActs.map((activity) => {
              const top = getTop(activity);
              if (top < 0) return null;
              const dayIndex = weekDays.findIndex((d) => d.toDateString() === dayStr);
              if (dayIndex < 0) return null;
              return (
                <div
                  key={activity.id}
                  className="absolute"
                  style={{
                    top,
                    height: getHeight(activity),
                    left: `calc(3rem + (100% - 3rem) / 7 * ${dayIndex})`,
                    width: `calc((100% - 3rem) / 7 - 4px)`,
                  }}
                >
                  <div className="mx-0.5 h-full overflow-hidden">
                    <ActivityCard
                      activity={activity}
                      reservationStatus={userReservationMap.get(activity.id)}
                      onClick={onActivityClick}
                      compact
                    />
                  </div>
                </div>
              );
            });
          })}
        </div>
      </div>
    </div>
  );
}

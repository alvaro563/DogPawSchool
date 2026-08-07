import { useMemo } from 'react';
import { cn } from '@/lib/utils';
import type { Activity } from '@/domain/entities/activity';
import { ActivityCard } from './activity-card';

interface MonthViewProps {
  currentDate: Date;
  activities: Activity[];
  userReservationMap: Map<number, string>;
  onActivityClick: (activity: Activity) => void;
}

const DAY_NAMES = ['Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb', 'Dom'];

function getMonthDays(year: number, month: number): Date[] {
  const firstDay = new Date(year, month, 1);
  const lastDay = new Date(year, month + 1, 0);
  const startPad = (firstDay.getDay() + 6) % 7;
  const days: Date[] = [];

  for (let i = startPad - 1; i >= 0; i--) {
    const d = new Date(year, month, -i);
    days.push(d);
  }

  for (let d = 1; d <= lastDay.getDate(); d++) {
    days.push(new Date(year, month, d));
  }

  const remaining = 7 - (days.length % 7);
  if (remaining < 7) {
    for (let d = 1; d <= remaining; d++) {
      days.push(new Date(year, month + 1, d));
    }
  }

  return days;
}

type ActivityMap = Map<string, Activity[]>;

export function MonthView({
  currentDate,
  activities,
  userReservationMap,
  onActivityClick,
}: MonthViewProps) {
  const today = new Date();
  const todayStr = today.toDateString();

  const activityMap: ActivityMap = useMemo(() => {
    const map = new Map<string, Activity[]>();
    for (const a of activities) {
      const dayKey = new Date(a.date).toDateString();
      const list = map.get(dayKey) || [];
      list.push(a);
      map.set(dayKey, list);
    }
    return map;
  }, [activities]);

  const days = useMemo(
    () => getMonthDays(currentDate.getFullYear(), currentDate.getMonth()),
    [currentDate],
  );

  const currentMonth = currentDate.getMonth();

  return (
    <div className="flex flex-1 flex-col">
      {/* Day headers */}
      <div className="grid grid-cols-7 border-b border-border">
        {DAY_NAMES.map((name) => (
          <div
            key={name}
            className="px-1 py-2 text-center text-xs font-medium text-muted-foreground"
          >
            <span className="hidden sm:inline">{name}</span>
            <span className="sm:hidden">{name.slice(0, 3)}</span>
          </div>
        ))}
      </div>

      {/* Day grid */}
      <div className="grid flex-1 grid-cols-7 auto-rows-fr">
        {days.map((day, i) => {
          const dayStr = day.toDateString();
          const dayActivities = activityMap.get(dayStr) || [];
          const isToday = dayStr === todayStr;
          const isOtherMonth = day.getMonth() !== currentMonth;

          return (
            <div
              key={i}
              className={cn(
                'flex flex-col border-b border-r border-border p-1 transition-colors hover:bg-muted/30',
                isOtherMonth && 'bg-muted/20',
              )}
            >
              <div className="mb-0.5 flex items-center justify-center">
                <span
                  className={cn(
                    'inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium',
                    isToday && 'bg-primary text-primary-foreground',
                    isOtherMonth && 'text-muted-foreground/50',
                  )}
                >
                  {day.getDate()}
                </span>
              </div>
              <div className="flex flex-col gap-0.5 overflow-hidden">
                {dayActivities.slice(0, 3).map((activity) => (
                  <ActivityCard
                    key={activity.id}
                    activity={activity}
                    reservationStatus={userReservationMap.get(activity.id)}
                    onClick={onActivityClick}
                    compact
                  />
                ))}
                {dayActivities.length > 3 && (
                  <p className="px-1 text-[10px] text-muted-foreground">
                    +{dayActivities.length - 3} más
                  </p>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

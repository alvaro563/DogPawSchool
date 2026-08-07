import { MapPin } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { Activity } from '@/domain/entities/activity';
import { formatActivityTime } from '@/features/calendar/hooks/use-calendar';

interface ActivityCardProps {
  activity: Activity;
  reservationStatus: string | undefined;
  onClick: (activity: Activity) => void;
  compact?: boolean;
}

export function ActivityCard({
  activity,
  reservationStatus,
  onClick,
  compact = false,
}: ActivityCardProps) {
  const isBooked = reservationStatus === 'CONFIRMED';
  const isPending = reservationStatus === 'PENDING_TO_CONFIRM';
  const isFull = activity.available_spots <= 0 && !isBooked;
  const isPast = activity.closed;

  const borderColor = isFull
    ? 'border-l-[3px] border-l-red-500 dark:border-l-red-400'
    : isBooked
      ? 'border-l-[3px] border-l-sky-500 bg-sky-50 dark:bg-sky-950/30'
      : isPending
        ? 'border-l-[3px] border-l-amber-500 bg-amber-50 dark:bg-amber-950/30'
        : 'border-l-[3px] border-l-primary/40';

  if (compact) {
    return (
      <button
        onClick={() => onClick(activity)}
        className={cn(
          'w-full rounded-md px-2 py-1 text-left transition-colors hover:bg-muted',
          borderColor,
          (isFull || isPast) && 'opacity-50',
        )}
      >
        <div className="truncate text-xs font-medium">{activity.name}</div>
        <div className="text-[10px] text-muted-foreground">
          {formatActivityTime(activity.date, activity.duration_in_hours)}
        </div>
      </button>
    );
  }

  return (
    <button
      onClick={() => onClick(activity)}
      className={cn(
        'flex w-full flex-col gap-0.5 rounded-lg px-3 py-2 text-left text-xs transition-colors hover:brightness-95',
        borderColor,
        (isFull || isPast) && 'opacity-50',
      )}
    >
      <div className="flex items-start justify-between gap-1">
        <span className="truncate font-semibold">{activity.name}</span>
        {isBooked && (
          <span className="shrink-0 rounded bg-sky-200 px-1.5 py-0.5 text-[10px] font-medium text-sky-800 dark:bg-sky-800 dark:text-sky-200">
            Reservado
          </span>
        )}
        {isPending && (
          <span className="shrink-0 rounded bg-amber-200 px-1.5 py-0.5 text-[10px] font-medium text-amber-800 dark:bg-amber-800 dark:text-amber-200">
            Pendiente
          </span>
        )}
        {isFull && !isBooked && (
          <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
            Completo
          </span>
        )}
      </div>
      <span className="text-muted-foreground">
        {formatActivityTime(activity.date, activity.duration_in_hours)}
      </span>
      <span className="flex items-center gap-1 text-muted-foreground">
        <MapPin className="h-3 w-3" />
        {activity.location}
      </span>
    </button>
  );
}

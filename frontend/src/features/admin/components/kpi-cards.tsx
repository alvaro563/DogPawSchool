import { CalendarDays, Clock, Dog, Ticket } from 'lucide-react';
import { useNavigate } from '@tanstack/react-router';
import { cn } from '@/lib/utils';

interface KpiCardsProps {
  todayClasses: number;
  pendingReservations: number;
  activeDogs: number;
  activePasses: number;
  isLoading: boolean;
}

const cards = [
  { key: 'classes', label: 'Clases hoy', icon: CalendarDays, color: 'text-sky-500', bg: 'bg-sky-50 dark:bg-sky-950/30', to: undefined },
  { key: 'pending', label: 'Pendientes', icon: Clock, color: 'text-amber-500', bg: 'bg-amber-50 dark:bg-amber-950/30', to: undefined },
  { key: 'dogs', label: 'Perros activos', icon: Dog, color: 'text-pink-500', bg: 'bg-pink-50 dark:bg-pink-950/30', to: '/active-dogs' },
  { key: 'passes', label: 'Bonos activos', icon: Ticket, color: 'text-emerald-500', bg: 'bg-emerald-50 dark:bg-emerald-950/30', to: undefined },
] as const;

export function KpiCards({ todayClasses, pendingReservations, activeDogs, activePasses, isLoading }: KpiCardsProps) {
  const navigate = useNavigate();

  const values: Record<string, number> = {
    classes: todayClasses,
    pending: pendingReservations,
    dogs: activeDogs,
    passes: activePasses,
  };

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      {cards.map(({ key, label, icon: Icon, color, bg, to }) => {
        const card = (
          <div
            className={cn(
              'flex flex-col gap-1 rounded-xl border border-border p-4 transition-shadow',
              to && 'cursor-pointer hover:shadow-md hover:border-primary/40',
              !to && 'hover:shadow-sm',
              bg,
            )}
          >
            <div className="flex items-center gap-2">
              <Icon className={cn('h-4 w-4', color)} strokeWidth={2} />
              <span className="text-xs font-medium text-muted-foreground">{label}</span>
            </div>
            <span className={cn('text-2xl font-bold', isLoading && 'animate-pulse text-muted-foreground')}>
              {isLoading ? '—' : values[key]}
            </span>
          </div>
        );

        if (to) {
          return (
            <button key={key} onClick={() => navigate({ to })} className="text-left">
              {card}
            </button>
          );
        }
        return <div key={key}>{card}</div>;
      })}
    </div>
  );
}

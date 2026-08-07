import { createLazyFileRoute } from '@tanstack/react-router';
import { CalendarView } from '@/features/calendar/components/calendar-view';

export const Route = createLazyFileRoute('/_authenticated/calendar')({
  component: CalendarView,
});

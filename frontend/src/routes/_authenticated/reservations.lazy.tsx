import { createLazyFileRoute } from '@tanstack/react-router';
import { ReservationListPage } from '@/features/reservations/components/reservation-list';

export const Route = createLazyFileRoute('/_authenticated/reservations')({
  component: ReservationListPage,
});

import { createFileRoute } from '@tanstack/react-router';
import { CancelledLateReservationsPage } from '@/features/admin/pages/cancelled-late-reservations-page';

export const Route = createFileRoute('/_authenticated/admin/reservations/cancelled-late')({
  component: CancelledLateReservationsPage,
});

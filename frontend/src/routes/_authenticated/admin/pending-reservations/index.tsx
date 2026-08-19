import { createFileRoute } from '@tanstack/react-router';
import { PendingReservationsPage } from '@/features/admin/pages/pending-reservations-page';

export const Route = createFileRoute('/_authenticated/admin/pending-reservations/')({
  component: PendingReservationsPage,
});

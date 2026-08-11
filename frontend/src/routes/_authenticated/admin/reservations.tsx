import { createFileRoute } from '@tanstack/react-router';
import { ReservationsManagementPage } from '@/features/admin/pages/reservations-management';

export const Route = createFileRoute('/_authenticated/admin/reservations')({
  component: ReservationsManagementPage,
});

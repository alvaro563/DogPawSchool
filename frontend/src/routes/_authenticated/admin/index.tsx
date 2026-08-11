import { createFileRoute } from '@tanstack/react-router';
import { AdminDashboard } from '@/features/admin/components/admin-dashboard';

export const Route = createFileRoute('/_authenticated/admin/')({
  component: AdminDashboard,
});

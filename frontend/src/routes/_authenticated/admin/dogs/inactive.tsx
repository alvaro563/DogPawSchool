import { createFileRoute } from '@tanstack/react-router';
import { InactiveDogsManagementPage } from '@/features/admin/pages/inactive-dogs-management';

export const Route = createFileRoute('/_authenticated/admin/dogs/inactive')({
  component: InactiveDogsManagementPage,
});

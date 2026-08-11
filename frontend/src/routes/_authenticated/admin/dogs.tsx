import { createFileRoute } from '@tanstack/react-router';
import { DogsManagementPage } from '@/features/admin/pages/dogs-management';

export const Route = createFileRoute('/_authenticated/admin/dogs')({
  component: DogsManagementPage,
});

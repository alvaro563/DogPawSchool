import { createLazyFileRoute } from '@tanstack/react-router';
import { DogsManagementPage } from '@/features/admin/pages/dogs-management';

export const Route = createLazyFileRoute('/_authenticated/active-dogs')({
  component: DogsManagementPage,
});

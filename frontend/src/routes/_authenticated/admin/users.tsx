import { createFileRoute } from '@tanstack/react-router';
import { UsersManagementPage } from '@/features/admin/pages/users-management';

export const Route = createFileRoute('/_authenticated/admin/users')({
  component: UsersManagementPage,
});

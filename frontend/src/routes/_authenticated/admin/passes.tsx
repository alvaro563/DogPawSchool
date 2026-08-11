import { createFileRoute } from '@tanstack/react-router';
import { PassesManagementPage } from '@/features/admin/pages/passes-management';

export const Route = createFileRoute('/_authenticated/admin/passes')({
  component: PassesManagementPage,
});

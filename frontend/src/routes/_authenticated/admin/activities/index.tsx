import { createFileRoute } from '@tanstack/react-router';
import { ActivitiesManagementPage } from '@/features/admin/pages/activities-management';

export const Route = createFileRoute('/_authenticated/admin/activities/')({
  component: ActivitiesManagementPage,
});

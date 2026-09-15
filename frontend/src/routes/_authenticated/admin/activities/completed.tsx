import { createFileRoute } from '@tanstack/react-router';
import { CompletedActivitiesPage } from '@/features/admin/pages/completed-activities-page';

export const Route = createFileRoute('/_authenticated/admin/activities/completed')({
  component: CompletedActivitiesPage,
});

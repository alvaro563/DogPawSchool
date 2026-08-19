import { createFileRoute } from '@tanstack/react-router';
import { TodayClassesPage } from '@/features/admin/pages/today-classes-page';

export const Route = createFileRoute('/_authenticated/admin/today-classes/')({
  component: TodayClassesPage,
});

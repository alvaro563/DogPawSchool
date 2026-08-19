import { createFileRoute } from '@tanstack/react-router';
import { ActivityDetailPage } from '@/features/admin/pages/activity-detail-page';

function ActivityDetailRoute() {
  const { id } = Route.useParams();
  return <ActivityDetailPage id={Number(id)} />;
}

export const Route = createFileRoute('/_authenticated/admin/activities/$id')({
  component: ActivityDetailRoute,
});

import { createLazyFileRoute } from '@tanstack/react-router';
import { DogDetailSheet } from '@/features/dogs/components/dog-detail-sheet';

function DogDetailPage() {
  const { id } = Route.useParams();

  return <DogDetailSheet dogId={Number(id)} onClose={() => window.history.back()} />;
}

export const Route = createLazyFileRoute('/_authenticated/dog-detail/$id')({
  component: DogDetailPage,
});

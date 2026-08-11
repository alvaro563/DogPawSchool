import { createLazyFileRoute } from '@tanstack/react-router';
import { DogListPage } from '@/features/dogs/components/dog-list';

export const Route = createLazyFileRoute('/_authenticated/dogs')({
  component: DogListPage,
});

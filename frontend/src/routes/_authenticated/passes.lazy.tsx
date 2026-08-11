import { createLazyFileRoute } from '@tanstack/react-router';
import { PassListPage } from '@/features/passes/components/pass-list';

export const Route = createLazyFileRoute('/_authenticated/passes')({
  component: PassListPage,
});

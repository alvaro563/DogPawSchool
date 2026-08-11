import { createLazyFileRoute } from '@tanstack/react-router';
import { IncompatibilitiesManagementPage } from '@/features/admin/pages/incompatibilities-management';

export const Route = createLazyFileRoute('/_authenticated/incompatibilities')({
  component: IncompatibilitiesManagementPage,
});

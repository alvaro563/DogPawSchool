import { createLazyFileRoute } from '@tanstack/react-router';
import { ProfilePage } from '@/features/profile/components/profile-page';

export const Route = createLazyFileRoute('/_authenticated/profile')({
  component: ProfilePage,
});

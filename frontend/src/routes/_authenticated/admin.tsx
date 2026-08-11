import { createFileRoute, Outlet, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_authenticated/admin')({
  beforeLoad: () => {
    const userRaw = localStorage.getItem('auth_user');
    if (!userRaw) {
      throw redirect({ to: '/auth/login' });
    }
    try {
      const user = JSON.parse(userRaw) as { role: string };
      if (user.role !== 'ADMIN') {
        throw redirect({ to: '/calendar' });
      }
    } catch {
      throw redirect({ to: '/auth/login' });
    }
  },
  component: () => <Outlet />,
});

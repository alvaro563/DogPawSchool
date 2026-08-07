import { createFileRoute, redirect } from '@tanstack/react-router';

function IndexRedirect() {
  return null;
}

export const Route = createFileRoute('/')({
  component: IndexRedirect,
  beforeLoad: () => {
    const token = localStorage.getItem('auth_token');
    const userRaw = localStorage.getItem('auth_user');

    if (!token || !userRaw) {
      throw redirect({ to: '/auth/login' });
    }

    try {
      const user = JSON.parse(userRaw) as { role: string };
      if (user.role === 'ADMIN') {
        throw redirect({ to: '/admin' });
      }
      throw redirect({ to: '/calendar' });
    } catch {
      throw redirect({ to: '/auth/login' });
    }
  },
});

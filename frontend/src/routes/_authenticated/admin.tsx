import { createFileRoute, redirect } from '@tanstack/react-router';
import { LayoutDashboard } from 'lucide-react';

function AdminPage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <div className="flex flex-col items-center justify-center space-y-4 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-foreground/10">
          <LayoutDashboard className="h-8 w-8 text-foreground" />
        </div>
        <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
          Panel de Administración
        </h1>
        <p className="max-w-md text-muted-foreground">
          Gestiona usuarios, perros, incompatibilidades, pases y reservas desde este panel.
        </p>
      </div>
    </div>
  );
}

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
  component: AdminPage,
});

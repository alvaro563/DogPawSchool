import { createRootRoute, Link, Outlet, useNavigate } from '@tanstack/react-router';
import { PawPrint, User, LogOut } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';

function RootLayout() {
  const { isAuthenticated, user, logout } = useAuth();
  const navigate = useNavigate();

  function handleLogout() {
    logout();
    navigate({ to: '/auth/login' });
  }

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <header className="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="flex h-16 w-full items-center justify-between px-4 sm:px-6 lg:px-8">
          <Link
            to="/"
            className="flex items-center gap-2 font-semibold tracking-tight transition-colors hover:opacity-80"
          >
            <PawPrint className="h-6 w-6 text-foreground" strokeWidth={2.5} />
            <span className="hidden sm:inline">DOG PAW BY ANA SUCH</span>
            <span className="sm:hidden">DogPaw</span>
          </Link>

          {isAuthenticated && user && (
            <div className="flex items-center gap-3 sm:gap-4">
              <div className="hidden items-center gap-2 sm:flex">
                <User className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm text-muted-foreground">
                  {user.name}
                </span>
                <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                  {user.role}
                </span>
              </div>
              <button
                onClick={handleLogout}
                className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                title="Cerrar sesión"
              >
                <LogOut className="h-4 w-4" />
                <span className="hidden sm:inline">Salir</span>
              </button>
            </div>
          )}
        </div>
      </header>

      <main className="flex-1">
        <Outlet />
      </main>
    </div>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});

import { useEffect, useState } from 'react';
import { createFileRoute, Outlet, useNavigate } from '@tanstack/react-router';
import { Menu } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Sidebar } from '@/components/layout/sidebar';
import { AdminModalProvider, useAdminModal } from '@/features/admin/hooks/admin-modal-context';
import { SelectedActivityProvider } from '@/features/calendar/hooks/selected-activity-context';
import { CreateActivityModal } from '@/features/admin/components/create-activity-modal';
import { AssignPassModal } from '@/features/admin/components/assign-pass-modal';
import { RegisterDogModal } from '@/features/admin/components/register-dog-modal';
import { RegisterClientModal } from '@/features/admin/components/register-client-modal';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetTitle } from '@/components/ui/sheet';

function AuthenticatedLayout() {
  const { isLoading, isAuthenticated, isAdmin } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const navigate = useNavigate();

  // Once the auth hook finishes bootstrapping (it asks /users/me
  // on mount) and we discover there is no session, redirect to
  // the login page. The previous design read localStorage here;
  // with cookie-based auth the SPA cannot read the token directly,
  // so we rely on the server telling us via /users/me. If that
  // call 401s (cookie expired, token_version bumped, etc.), the
  // hook clears the local cache and sets user=null, and this
  // effect fires once isLoading flips to false.
  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate({ to: '/auth/login' });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading || !isAuthenticated) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  return (
    <AdminModalProvider>
      <SelectedActivityProvider>
        <div className="flex min-h-[calc(100vh-4rem)]">
          <aside className="hidden w-64 shrink-0 border-r border-border lg:block">
            <Sidebar />
          </aside>

          <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
            <Button
              variant="outline"
              size="icon"
              className="fixed bottom-4 left-4 z-50 h-12 w-12 rounded-full shadow-lg lg:hidden"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu className="h-5 w-5" />
            </Button>
            <SheetContent side="left" className="w-72 p-0 pt-10">
              <SheetTitle className="sr-only">Navegación</SheetTitle>
              <Sidebar onNavigate={() => setSidebarOpen(false)} />
            </SheetContent>
          </Sheet>

          <main className="flex-1 overflow-auto">
            <Outlet />
          </main>

          {isAdmin && <GlobalModals />}
        </div>
      </SelectedActivityProvider>
    </AdminModalProvider>
  );
}

function GlobalModals() {
  const { active, close } = useAdminModal();

  return (
    <>
      <CreateActivityModal open={active === 'activity'} onOpenChange={(o) => { if (!o) close(); }} />
      <AssignPassModal open={active === 'pass'} onOpenChange={(o) => { if (!o) close(); }} />
      <RegisterDogModal open={active === 'dog'} onOpenChange={(o) => { if (!o) close(); }} />
      <RegisterClientModal open={active === 'client'} onOpenChange={(o) => { if (!o) close(); }} />
    </>
  );
}

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: () => {
    // Cookie-based auth: we cannot synchronously check the session
    // here (the token is HttpOnly and the SPA cannot read it). Let
    // the AuthProvider's /users/me bootstrap resolve the truth.
    // Individual handlers will 401 any actual expired session and
    // the http-client refresh interceptor will handle renewals.
  },
  component: AuthenticatedLayout,
});

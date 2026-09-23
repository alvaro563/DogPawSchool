import { createFileRoute, Outlet } from '@tanstack/react-router';

export const Route = createFileRoute('/_authenticated/admin')({
  // Role enforcement now happens server-side (AdminRequired
  // middleware rejects non-admin calls with 403). The previous
  // localStorage check was a UX hint at best; with cookies the
  // SPA cannot read the role directly, and the /users/me bootstrap
  // is async, so we rely on the server to gate this layout.
  component: () => <Outlet />,
});

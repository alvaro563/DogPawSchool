import { createFileRoute, redirect } from '@tanstack/react-router';

// The previous design gated this route on the contents of
// localStorage. With cookie-based auth the SPA never has direct
// access to the token — it can only ask the server. We let the
// AuthProvider's useEffect kick off the /users/me bootstrap; while
// that is in flight, fall through to /calendar (the most common
// landing for an authenticated regular user). The authenticated
// route guards then verify the real session.
function IndexRedirect() {
  return null;
}

export const Route = createFileRoute('/')({
  component: IndexRedirect,
  beforeLoad: () => {
    // We do not have synchronous access to the auth state here
    // (beforeLoad runs before the AuthProvider mounts). Best
    // guess: send the user to /calendar if they previously
    // landed on this SPA; the route guards will redirect them
    // to /auth/login if their cookie is dead.
    throw redirect({ to: '/calendar' });
  },
});

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import type { User, UserRole } from '@/domain/entities/user';
import type { LoginInput } from '@/domain/schemas/auth-schema';
import { AuthRepositoryImpl } from '@/infrastructure/repositories/auth-repository.impl';
import { userStorage } from '@/infrastructure/storage/user';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isLoading: boolean;
  login: (input: LoginInput) => Promise<User>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

const authRepository = new AuthRepositoryImpl();

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Bootstrap: ask the server who I am. The access_token cookie
  // travels automatically. If 401, no session — render the auth
  // shell. If 200, hydrate the user from the response body.
  //
  // The previous design read localStorage here. localStorage is
  // XSS-readable; that was the root cause of the session-security
  // audit. The new design relies on the HttpOnly cookie + a single
  // /users/me round-trip on first load.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      // Try the cached user first to avoid a blank flash. If the
      // cookie is still valid the server will accept this; if not,
      // /users/me returns 401 and we clear the cache.
      const cached = userStorage.get();
      if (cached) {
        setUser(cached);
      }
      try {
        const { user: fresh } = await authRepository.me();
        if (!cancelled) {
          if (fresh) {
            userStorage.set(fresh);
            setUser(fresh);
          } else {
            // 200 but no user in the envelope — a shape mismatch or
            // corrupt payload. Treat as "no session": clearing the
            // state lets the authenticated-route guard redirect to
            // /auth/login instead of rendering the shell with an
            // undefined user (the reload-lockout bug).
            userStorage.remove();
            setUser(null);
          }
        }
      } catch {
        if (!cancelled) {
          userStorage.remove();
          setUser(null);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (input: LoginInput): Promise<User> => {
    // Login sets the HttpOnly cookies in the browser; the SPA
    // receives only the user profile.
    const response = await authRepository.login(input);
    userStorage.set(response.user);
    setUser(response.user);
    return response.user;
  }, []);

  // logout hits POST /auth/logout (server clears cookies) and
  // always clears local state, even if the server call fails —
  // a failed logout must not leave the SPA believing the user is
  // still authenticated.
  const logout = useCallback(async () => {
    try {
      await authRepository.logout();
    } catch {
      // Server unreachable: the cookies may already be expired or
      // never set; clear local state anyway.
    }
    userStorage.remove();
    setUser(null);
  }, []);

  const value = useMemo<AuthState>(() => ({
    user,
    // Nullish check, not `!== null`: if state ever holds undefined
    // (a response-shape bug), `undefined !== null` would be true and
    // the app would treat a userless session as authenticated —
    // rendering the shell without a header user and skipping the
    // login redirect. With `!= null` both null and undefined mean
    // "not authenticated".
    isAuthenticated: user != null,
    isAdmin: user?.role === 'ADMIN',
    isLoading,
    login,
    logout,
  }), [user, isLoading, login, logout]);

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}

export function isAdminRole(role: UserRole): boolean {
  return role === 'ADMIN';
}

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
import storageToken from '@/infrastructure/storage/token';
import { userStorage } from '@/infrastructure/storage/user';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isLoading: boolean;
  login: (input: LoginInput) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

const authRepository = new AuthRepositoryImpl();

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const token = storageToken.get();
    const savedUser = userStorage.get();
    if (token && savedUser) {
      setUser(savedUser);
    }
    setIsLoading(false);
  }, []);

  const login = useCallback(async (input: LoginInput) => {
    const response = await authRepository.login(input);
    storageToken.set(response.token);
    userStorage.set(response.user);
    setUser(response.user);
  }, []);

  const logout = useCallback(() => {
    storageToken.remove();
    userStorage.remove();
    setUser(null);
  }, []);

  const value = useMemo<AuthState>(() => ({
    user,
    isAuthenticated: user !== null,
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

const USER_KEY = 'auth_user';
import type { User } from '@/domain/entities/user';

class UserStorage {
  get(): User | null {
    const raw = localStorage.getItem(USER_KEY);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as User;
    } catch {
      return null;
    }
  }

  set(user: User): void {
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  }

  remove(): void {
    localStorage.removeItem(USER_KEY);
  }
}

export const userStorage = new UserStorage();

import type { User } from '@/domain/entities/user';

const USER_KEY = 'auth_user';

// UserStorage caches the user profile in localStorage to avoid a
// blank flash on first page load while /users/me is in flight. The
// server is the source of truth — if the cached user disagrees with
// what /users/me returns, the cached copy is overwritten. Crucially,
// THIS STORES NO SECRETS: no tokens, no password hashes, just the
// public profile fields (id, email, name, role). An XSS that reads
// localStorage can already impersonate the user via the cookie.
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
    // Never persist a falsy user: JSON.stringify(undefined) returns
    // undefined (the value), and localStorage would store the literal
    // string "undefined", which parses back as garbage and poisoned
    // the reload bootstrap once. Clear instead.
    if (!user) {
      this.remove();
      return;
    }
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  }

  remove(): void {
    localStorage.removeItem(USER_KEY);
  }
}

export const userStorage = new UserStorage();

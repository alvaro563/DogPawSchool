const TOKEN_KEY = 'auth_token';

class StorageToken {
  get(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  }

  set(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
  }

  remove(): void {
    localStorage.removeItem(TOKEN_KEY);
  }
}

const storageToken = new StorageToken();
export default storageToken;

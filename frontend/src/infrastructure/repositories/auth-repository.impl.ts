import type {
  AuthRepository,
  AuthResponse,
  ChangePasswordInput,
  ChangePasswordResponse,
} from '@/domain/repositories/auth-repository';
import type { LoginInput, RegisterInput } from '@/domain/schemas/auth-schema';
import apiClient from '@/infrastructure/api/http-client';

export class AuthRepositoryImpl implements AuthRepository {
  async login(data: LoginInput): Promise<AuthResponse> {
    // 401 from /auth/login means wrong credentials — a domain
    // rejection the LoginForm renders inline, NOT a dead session.
    // Never let the http-client hard-redirect (full reload) here.
    return apiClient.post<AuthResponse>('/auth/login', data, {
      skipAuthRedirect: true,
    });
  }

  async registerWithInvitation(data: RegisterInput): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>('/auth/register', data, {
      skipAuthRedirect: true,
    });
  }

  async changePassword(data: ChangePasswordInput): Promise<ChangePasswordResponse> {
    // PATCH /auth/password can return 401 with body
    // `{ "error": "invalid_credentials" }` when the OLD password is
    // wrong. That 401 is a domain-level rejection (the user mistyped),
    // NOT a session-expired signal — we MUST NOT let the http-client
    // auto-redirect to /auth/login mid-form. `skipAuthRedirect: true`
    // surfaces the ApiError to the caller so the form can render the
    // error inline.
    return apiClient.patch<ChangePasswordResponse>(
      '/auth/password',
      data,
      { skipAuthRedirect: true },
    );
  }

  async logout(): Promise<void> {
    // Fire-and-forget: even if the server call fails (including 401
    // when the session is already dead), the cookies are cleared on
    // the next response, and the SPA's local state is cleared by the
    // caller regardless of this outcome. A 401 here must not trigger
    // a hard redirect either — the caller is already navigating to
    // the login page.
    await apiClient.post<void>('/auth/logout', undefined, {
      skipAuthRedirect: true,
    });
  }

  async me(): Promise<{ user: AuthResponse['user'] }> {
    // GET /users/me hydrates the SPA's user state from the access
    // cookie. The cookie travels automatically thanks to
    // credentials: 'include' in the http-client. Used on first
    // page load to decide whether to render the auth shell.
    //
    // skipAuthRedirect: true — the bootstrap runs on EVERY page
    // load, including on /auth/login itself. Without this, a 401
    // (no/invalid session) would assign window.location.href and
    // reload the document, which remounts AuthProvider, which
    // calls me() again → infinite reload loop. The AuthProvider
    // catch already handles the no-session case by setting
    // user = null; no navigation needed. The refresh attempt in
    // the http-client still runs (decoupled from this flag), so
    // an expired-access/valid-refresh session renews silently.
    return apiClient.get<{ user: AuthResponse['user'] }>('/users/me', undefined, {
      skipAuthRedirect: true,
    });
  }
}

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
    return apiClient.post<AuthResponse>('/auth/login', data);
  }

  async registerWithInvitation(data: RegisterInput): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>('/auth/register', data);
  }

  async changePassword(data: ChangePasswordInput): Promise<ChangePasswordResponse> {
    // PATCH /auth/password returns 401 with body
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
}

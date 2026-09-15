import type { AuthRepository, AuthResponse } from '@/domain/repositories/auth-repository';
import type { LoginInput, RegisterInput } from '@/domain/schemas/auth-schema';
import apiClient from '@/infrastructure/api/http-client';

export class AuthRepositoryImpl implements AuthRepository {
  async login(data: LoginInput): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>('/auth/login', data);
  }

  async registerWithInvitation(data: RegisterInput): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>('/auth/register', data);
  }
}

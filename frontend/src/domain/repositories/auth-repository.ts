import type { LoginInput } from '@/domain/schemas/auth-schema';
import type { User } from '@/domain/entities/user';

export interface AuthResponse {
  token: string;
  user: User;
}

export interface AuthRepository {
  login(data: LoginInput): Promise<AuthResponse>;
}

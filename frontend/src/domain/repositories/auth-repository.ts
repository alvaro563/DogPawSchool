import type { LoginInput, RegisterInput } from '@/domain/schemas/auth-schema';
import type { User } from '@/domain/entities/user';

export interface AuthResponse {
  token: string;
  user: User;
}

// ChangePasswordInput is the user-facing payload for PATCH /auth/password.
// The backend requires the old password (re-authentication step) and
// the new password in plain text — both are validated server-side
// (8-72 chars, old verified against the stored hash). The frontend
// runs the same validation client-side for fast feedback, but the
// backend is the source of truth.
export interface ChangePasswordInput {
  old_password: string;
  new_password: string;
}

// ChangePasswordResponse is the (deliberately empty) success envelope.
// The backend uses an empty body + 200 OK to signal success; this
// shape only exists so callers can `await changePassword(...)` with
// a typed return.
export interface ChangePasswordResponse {
  message: string;
}

export interface AuthRepository {
  login(data: LoginInput): Promise<AuthResponse>;
  registerWithInvitation(data: RegisterInput): Promise<AuthResponse>;
  changePassword(data: ChangePasswordInput): Promise<ChangePasswordResponse>;
}

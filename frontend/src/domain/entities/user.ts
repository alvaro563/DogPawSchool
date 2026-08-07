export type UserRole = 'ADMIN' | 'REGULAR';

export interface User {
  id: number;
  name: string;
  email: string;
  role: UserRole;
  is_active: boolean;
}

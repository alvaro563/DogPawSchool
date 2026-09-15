import type { User } from '@/domain/entities/user';
import apiClient from '@/infrastructure/api/http-client';

interface UserListResponse {
  users: User[];
  limit: number;
  offset: number;
  count: number;
}

export async function fetchAllUsers(): Promise<User[]> {
  const data = await apiClient.get<UserListResponse>('/users', { limit: '200' });
  return data.users;
}

export async function fetchUserByID(id: number): Promise<User> {
  return apiClient.get<User>(`/users/${id}`);
}

export async function updateUser(id: number, patch: { name?: string; email?: string }): Promise<{ id: number }> {
  return apiClient.patch<{ id: number }>(`/users/${id}`, patch);
}

export async function deactivateUser(id: number): Promise<{ id: number; is_active: boolean }> {
  return apiClient.post<{ id: number; is_active: boolean }>(`/users/${id}/deactivate`);
}

export async function activateUser(id: number): Promise<{ id: number; is_active: boolean }> {
  return apiClient.post<{ id: number; is_active: boolean }>(`/users/${id}/activate`);
}

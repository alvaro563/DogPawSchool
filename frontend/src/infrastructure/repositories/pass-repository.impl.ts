import type { Pass } from '@/domain/entities/pass';
import type { PassListResponse } from '@/domain/entities/pass';
import apiClient from '@/infrastructure/api/http-client';

interface CreatePassRequest {
  num_of_sessions: number;
  price: number;
  pass_type: string;
  expires_at?: string;
}

export async function fetchPassesByUser(userId: number): Promise<Pass[]> {
  const data = await apiClient.get<PassListResponse>(`/users/${userId}/passes`, {
    limit: '100',
  });
  return data.passes;
}

export async function fetchAllPasses(): Promise<Pass[]> {
  const data = await apiClient.get<PassListResponse>('/passes', { limit: '200' });
  return data.passes;
}

export async function createPass(
  userId: number,
  body: CreatePassRequest,
): Promise<{ id: number }> {
  return apiClient.post<{ id: number }>(`/users/${userId}/passes`, body);
}

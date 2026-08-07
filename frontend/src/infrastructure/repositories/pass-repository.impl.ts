import type { Pass } from '@/domain/entities/pass';
import type { PassListResponse } from '@/domain/entities/pass';
import apiClient from '@/infrastructure/api/http-client';

export async function fetchPassesByUser(userId: number): Promise<Pass[]> {
  const data = await apiClient.get<PassListResponse>(`/users/${userId}/passes`, {
    limit: '100',
  });
  return data.passes;
}

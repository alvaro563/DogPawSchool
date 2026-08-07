import type { Dog } from '@/domain/entities/dog';
import type { DogListResponse } from '@/domain/entities/dog';
import apiClient from '@/infrastructure/api/http-client';

export async function fetchDogsByOwner(ownerId: number): Promise<Dog[]> {
  const data = await apiClient.get<DogListResponse>(`/dogs/owner/${ownerId}`, {
    limit: '100',
  });
  return data.dogs;
}

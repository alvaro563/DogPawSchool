import type { Dog } from '@/domain/entities/dog';
import type { DogListResponse } from '@/domain/entities/dog';
import apiClient from '@/infrastructure/api/http-client';

interface RegisterDogRequest {
  name: string;
  breed: string;
  age_in_months: number;
  sex: string;
  weight_kg: number;
  passport: string;
  user_id: number;
}

interface RegisterDogResponse {
  id: number;
}

export async function fetchDogsByOwner(ownerId: number): Promise<Dog[]> {
  const data = await apiClient.get<DogListResponse>(`/dogs/owner/${ownerId}`, {
    limit: '100',
  });
  return data.dogs;
}

export async function fetchAllDogs(): Promise<Dog[]> {
  const data = await apiClient.get<DogListResponse>('/dogs', { limit: '100' });
  return data.dogs;
}

export async function fetchActiveDogsCount(): Promise<number> {
  const data = await apiClient.get<DogListResponse>('/dogs/active', { limit: '10000' });
  return data.dogs.length;
}

export async function fetchAllActiveDogs(): Promise<Dog[]> {
  const data = await apiClient.get<DogListResponse>('/dogs/active', { limit: '10000' });
  return data.dogs;
}

export async function fetchDogsByNeutered(): Promise<Dog[]> {
  const data = await apiClient.get<DogListResponse>('/dogs/neutered/true', { limit: '10000' });
  return data.dogs;
}

export async function fetchDogsByHeat(): Promise<Dog[]> {
  const data = await apiClient.get<DogListResponse>('/dogs/heat/true', { limit: '10000' });
  return data.dogs;
}

export async function fetchDogByID(id: number): Promise<Dog> {
  return apiClient.get<Dog>(`/dogs/${id}`);
}

export async function updateDog(id: number, patch: Record<string, unknown>): Promise<{ id: number }> {
  return apiClient.patch<{ id: number }>(`/dogs/${id}`, patch);
}

export async function registerDog(body: RegisterDogRequest): Promise<RegisterDogResponse> {
  return apiClient.post<RegisterDogResponse>('/dogs', body);
}

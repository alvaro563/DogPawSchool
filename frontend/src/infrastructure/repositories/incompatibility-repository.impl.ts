import type { IncompatibilityDTO } from '@/domain/entities/dog';
import apiClient from '@/infrastructure/api/http-client';

interface IncompatibilityListResponse {
  incompatibilities: IncompatibilityDTO[];
  count: number;
}

interface CreateIncompatibilityRequest {
  name: string;
  level: string;
  code?: string;
  target_trait_code?: string;
}

export async function fetchAllIncompatibilities(): Promise<IncompatibilityDTO[]> {
  const data = await apiClient.get<IncompatibilityListResponse>('/incompatibilities');
  return data.incompatibilities;
}

export async function createIncompatibility(
  body: CreateIncompatibilityRequest,
): Promise<{ id: number }> {
  return apiClient.post<{ id: number }>('/incompatibilities', body);
}

export async function addTraitToDog(
  dogId: number,
  traitId: number,
): Promise<unknown> {
  return apiClient.post(`/dogs/${dogId}/traits`, { trait_id: traitId });
}

export async function addTriggerToDog(
  dogId: number,
  triggerId: number,
): Promise<unknown> {
  return apiClient.post(`/dogs/${dogId}/incompatibilities`, { trigger_id: triggerId });
}

export async function removeIncompatibilityFromDog(
  dogId: number,
  incompatId: number,
): Promise<unknown> {
  return apiClient.delete(`/dogs/${dogId}/incompatibilities/${incompatId}`);
}

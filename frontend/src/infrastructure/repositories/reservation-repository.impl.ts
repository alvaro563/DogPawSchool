import type { ReservationView } from '@/domain/entities/reservation';
import type { ReservationListResponse } from '@/domain/entities/reservation';
import type { CreateReservationRequest, CreateReservationResponse } from '@/domain/entities/reservation';
import apiClient from '@/infrastructure/api/http-client';

export async function fetchUserReservations(
  userId: number,
  status?: string,
): Promise<ReservationView[]> {
  const params: Record<string, string> = { limit: '100' };
  if (status) {
    params.status = status;
  }
  const data = await apiClient.get<ReservationListResponse>(
    `/users/${userId}/reservations`,
    params,
  );
  return data.reservations;
}

export async function createReservation(
  userId: number,
  body: CreateReservationRequest,
): Promise<CreateReservationResponse> {
  return apiClient.post<CreateReservationResponse>(`/users/${userId}/reservations`, body);
}

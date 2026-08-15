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

export async function confirmReservation(
  userId: number,
  reservationId: number,
): Promise<{ id: number; status: string }> {
  return apiClient.post<{ id: number; status: string }>(
    `/users/${userId}/reservations/${reservationId}/confirm`,
  );
}

export async function rejectReservation(
  userId: number,
  reservationId: number,
): Promise<{ id: number; status: string }> {
  return apiClient.post<{ id: number; status: string }>(
    `/users/${userId}/reservations/${reservationId}/reject`,
  );
}

export async function fetchAllReservations(): Promise<ReservationView[]> {
  const data = await apiClient.get<ReservationListResponse>('/reservations', { limit: '200' });
  return data.reservations;
}

export async function fetchUpcomingReservations(): Promise<ReservationView[]> {
  const data = await apiClient.get<ReservationListResponse>('/reservations/upcoming', { limit: '200' });
  return data.reservations;
}

export async function createAdminReservation(
  body: CreateReservationRequest,
): Promise<CreateReservationResponse> {
  return apiClient.post<CreateReservationResponse>('/reservations', body);
}

import type { ReservationView } from '@/domain/entities/reservation';
import type { ReservationListResponse } from '@/domain/entities/reservation';
import type { CreateReservationRequest, CreateReservationResponse } from '@/domain/entities/reservation';
import type { ActivityRoster } from '@/domain/entities/reservation';
import type { PendingReservationsResponse } from '@/domain/entities/reservation';
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

// cancelReservation cancels a CONFIRMED reservation for the
// currently authenticated user. The endpoint is the same one the
// user has always called: /users/{user_id}/reservations/{id}/cancel.
// The user_id MUST match the caller's user_id — the backend
// enforces ownership (dog + pass both belong to user_id).
export async function cancelReservation(
  userId: number,
  reservationId: number,
): Promise<{ id: number; status: string }> {
  return apiClient.post<{ id: number; status: string }>(
    `/users/${userId}/reservations/${reservationId}/cancel`,
  );
}

// cancelReservationAdmin is the admin-cancel endpoint. It does
// NOT take user_id in the path because the admin does not need
// to know who owns the reservation. Backend uses
// NewCancelReservationAdminInput so dog/pass ownership checks
// are bypassed. The route sits under the admin group in
// cmd/api/router.go, so the auth middleware enforces admin role.
export async function cancelReservationAdmin(
  reservationId: number,
): Promise<{ id: number; status: string }> {
  return apiClient.post<{ id: number; status: string }>(
    `/reservations/${reservationId}/cancel`,
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

// fetchActivityRoster returns the admin class-day view for a single
// activity: the activity itself plus confirmed/pending attendees
// already partitioned by status and annotated with owner name. Used
// by the "Clases hoy" dashboard drill-down.
//
// Backend note: the router wires this endpoint under the `admin`
// Gin group, but that group has NO `/admin` URL prefix (it is
// declared as `v1.Group("")`). The path is therefore
// `/activities/{id}/roster`, NOT `/admin/activities/{id}/roster`.
// The `/admin` prefix only exists in the client-side TanStack
// Router (page URLs), not in the API. Keep this in sync with
// cmd/api/router.go.
export async function fetchActivityRoster(activityId: number): Promise<ActivityRoster> {
  return apiClient.get<ActivityRoster>(`/activities/${activityId}/roster`);
}

// fetchPendingReservations returns every reservation awaiting
// admin approval, each annotated with the dog owner's id and
// name. Server-side partitioning keeps the client thin: one fetch
// renders the full triage page. Used by the "Pendientes" dashboard
// drill-down.
//
// Same backend note as fetchActivityRoster: the `admin` Gin group
// has no `/admin` URL prefix, so the path is `/reservations/pending`.
export async function fetchPendingReservations(
  limit = 100,
  offset = 0,
): Promise<PendingReservationsResponse> {
  return apiClient.get<PendingReservationsResponse>('/reservations/pending', {
    limit: String(limit),
    offset: String(offset),
  });
}

import type { Activity } from '@/domain/entities/activity';
import type { ActivityListResponse } from '@/domain/entities/activity';
import apiClient from '@/infrastructure/api/http-client';

export async function fetchActivities(
  from: string,
  to: string,
  closed?: boolean,
): Promise<Activity[]> {
  const params: Record<string, string> = {
    limit: '100',
    from,
    to,
  };
  // Omit the param entirely when closed is undefined so the
  // backend returns both open and closed (today-classes relies
  // on this).
  if (closed !== undefined) {
    params.closed = String(closed);
  }
  const data = await apiClient.get<ActivityListResponse>('/activities', params);
  return data.activities;
}

export async function fetchActivityByID(id: number): Promise<Activity> {
  return apiClient.get<Activity>(`/activities/${id}`);
}

// bulkCompleteActivity marks every CONFIRMED reservation of the
// activity as COMPLETED AND closes the activity, in a single
// transaction. Returns { id, completed, closed } where `completed`
// is the number of reservations that transitioned in this call (0
// is valid when the activity is already closed or has no CONFIRMED
// rows) and `closed` is always true on success.
export async function bulkCompleteActivity(
  activityId: number,
): Promise<{ id: number; completed: number; closed: boolean }> {
  return apiClient.post<{ id: number; completed: number; closed: boolean }>(
    `/activities/${activityId}/complete-all`,
  );
}

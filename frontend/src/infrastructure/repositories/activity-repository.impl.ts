import type { Activity } from '@/domain/entities/activity';
import type { ActivityListResponse } from '@/domain/entities/activity';
import apiClient from '@/infrastructure/api/http-client';

export async function fetchActivities(from: string, to: string): Promise<Activity[]> {
  const params: Record<string, string> = {
    limit: '100',
    from,
    to,
  };
  const data = await apiClient.get<ActivityListResponse>('/activities', params);
  return data.activities;
}

export async function fetchActivityByID(id: number): Promise<Activity> {
  return apiClient.get<Activity>(`/activities/${id}`);
}

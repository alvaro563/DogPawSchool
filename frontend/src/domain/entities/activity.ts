export type ActivityType = 'SOCIALIZATION_GROUP' | 'ROUTE' | 'INDIVIDUAL_CLASS' | 'EXTRA';

export interface Activity {
  id: number;
  name: string;
  description: string;
  activity_type: ActivityType;
  location: string;
  max_capacity: number;
  available_spots: number;
  duration_in_hours: number;
  date: string;
  closed: boolean;
}

export interface ActivityListResponse {
  activities: Activity[];
  limit: number;
  offset: number;
  count: number;
}

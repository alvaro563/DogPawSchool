import type { Activity } from '@/domain/entities/activity';

export interface ActivityRepository {
  listByDateRange(from: string, to: string): Promise<Activity[]>;
  getByID(id: number): Promise<Activity>;
}

import type { Pass } from '@/domain/entities/pass';

export interface PassRepository {
  listByUser(userId: number): Promise<Pass[]>;
}

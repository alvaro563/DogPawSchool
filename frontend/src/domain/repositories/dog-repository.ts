import type { Dog } from '@/domain/entities/dog';

export interface DogRepository {
  listByOwner(ownerId: number): Promise<Dog[]>;
}

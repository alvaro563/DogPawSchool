import type { Dog } from '@/domain/entities/dog';
import type { DogEditInput } from '@/domain/schemas/dog-schema';

export interface DogRepository {
  listByOwner(ownerId: number): Promise<Dog[]>;
  getByID(id: number): Promise<Dog>;
  update(id: number, patch: Partial<DogEditInput>): Promise<void>;
}

export type { Dog };

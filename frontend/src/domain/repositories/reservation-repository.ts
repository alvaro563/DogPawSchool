import type { ReservationView, CreateReservationRequest, CreateReservationResponse } from '@/domain/entities/reservation';

export interface ReservationRepository {
  listByUser(userId: number, status?: string): Promise<ReservationView[]>;
  create(userId: number, data: CreateReservationRequest): Promise<CreateReservationResponse>;
}

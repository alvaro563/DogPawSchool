export type ReservationStatus =
  | 'CONFIRMED'
  | 'COMPLETED'
  | 'CANCELLED_IN_TIME'
  | 'CANCELLED_LATE'
  | 'FORGIVEN'
  | 'NO_SHOW'
  | 'PENDING_TO_CONFIRM';

export interface ReservationView {
  id: number;
  status: ReservationStatus;
  created_at: string;
  activity_id: number;
  activity_name: string;
  activity_date: string;
  activity_location: string;
  activity_closed: boolean;
  dog_id: number;
  dog_name: string;
  pass_id: number;
  pass_type: string;
  pass_remaining: number;
}

export interface ReservationListResponse {
  reservations: ReservationView[];
  limit: number;
  offset: number;
  count: number;
}

export interface CreateReservationRequest {
  activity_id: number;
  dog_id: number;
  pass_id: number;
}

export interface CreateReservationResponse {
  id: number;
  status: string;
}

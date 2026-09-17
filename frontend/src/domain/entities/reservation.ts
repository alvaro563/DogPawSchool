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
  owner_id: number;
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
  // Spanish explanations of why the reservation was held in
  // StatusPendingToConfirm. Populated only when status is
  // 'PENDING_TO_CONFIRM' and the backend attached reasons; absent on
  // CONFIRMED responses.
  pending_reasons?: string[];
}

// ActivityRoster is the admin class-day view returned by
// GET /admin/activities/{id}/roster. Server-side partitioning:
// confirmed and pending are already separated so the client can
// render the page with one fetch and zero filtering.
export interface ActivityRosterEntry {
  reservation_id: number;
  dog_id: number;
  dog_name: string;
  owner_id: number;
  owner_name: string;
}

export interface ActivityRoster {
  activity: import('./activity').Activity;
  confirmed: ActivityRosterEntry[];
  pending: ActivityRosterEntry[];
}

// PendingReservationEntry is the admin "pending to approve" row.
// It mirrors ActivityRosterEntry for the activity-scoped view but
// adds activity_date and activity_location because the global
// pending page needs to render the class context per row.
export interface PendingReservationEntry {
  reservation_id: number;
  dog_id: number;
  dog_name: string;
  owner_id: number;
  owner_name: string;
  activity_id: number;
  activity_name: string;
  activity_date: string;
  activity_location: string;
}

export interface PendingReservationsResponse {
  pending: PendingReservationEntry[];
  limit: number;
  offset: number;
  count: number;
}

export interface Pass {
  id: number;
  num_of_sessions: number;
  remaining_sessions: number;
  price: number;
  pass_type: string;
  user_id: number;
  created_at: string;
  updated_at: string;
  expires_at: string | null;
}

export interface PassListResponse {
  passes: Pass[];
  limit: number;
  offset: number;
  count: number;
}

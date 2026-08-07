export interface Dog {
  id: number;
  name: string;
  breed: string;
  age_in_months: number;
  sex: string;
  neutered: boolean;
  heat: boolean;
  weight_kg: number;
  photo_url: string;
  passport: string;
  user_id: number;
  is_active: boolean;
}

export interface DogListResponse {
  dogs: Dog[];
  limit: number;
  offset: number;
  count: number;
}

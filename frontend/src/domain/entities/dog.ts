export interface IncompatibilityDTO {
  id: number;
  name: string;
  level: string;
  code?: string;
  target_trait_code?: string;
}

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
  medical_notes: string;
  educator_notes: string;
  passport: string;
  user_id: number;
  owner_name: string;
  is_active: boolean;
  traits: IncompatibilityDTO[];
  incompatibilities: IncompatibilityDTO[];
}

export interface DogListResponse {
  dogs: Dog[];
  limit: number;
  offset: number;
  count: number;
}

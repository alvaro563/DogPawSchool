import { z } from 'zod';

export const dogEditSchema = z.object({
  name: z.string().min(1, 'Nombre obligatorio'),
  breed: z.string().min(1, 'Raza obligatoria'),
  age_in_months: z.number().int().min(1, 'Edad debe ser positiva'),
  sex: z.enum(['MALE', 'FEMALE']),
  neutered: z.boolean(),
  heat: z.boolean(),
  weight_kg: z.number().min(0.1, 'Peso debe ser positivo'),
  photo_url: z.string(),
  medical_notes: z.string(),
  educator_notes: z.string(),
  passport: z.string().min(1, 'Pasaporte obligatorio'),
  is_active: z.boolean(),
  has_special_condition: z.boolean(),
});

export type DogEditInput = z.infer<typeof dogEditSchema>;

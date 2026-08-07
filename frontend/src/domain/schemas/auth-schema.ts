import { z } from 'zod';

export const loginSchema = z.object({
  email: z
    .string()
    .min(1, 'Email es obligatorio')
    .email('Email no es válido'),
  password: z
    .string()
    .min(1, 'Contraseña es obligatoria'),
});

export type LoginInput = z.infer<typeof loginSchema>;

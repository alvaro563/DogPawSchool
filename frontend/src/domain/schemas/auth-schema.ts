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

export const registerSchema = z.object({
  token: z.string().min(1, 'Token requerido'),
  name: z
    .string()
    .trim()
    .min(1, 'Nombre es obligatorio')
    .max(100, 'Máximo 100 caracteres'),
  password: z
    .string()
    .min(8, 'Mínimo 8 caracteres')
    .max(72, 'Máximo 72 caracteres'),
});

export type RegisterInput = z.infer<typeof registerSchema>;

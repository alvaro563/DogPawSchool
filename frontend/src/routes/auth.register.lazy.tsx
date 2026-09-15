import { createLazyFileRoute } from '@tanstack/react-router';
import { RegisterForm } from '@/features/auth/pages/register-form';

export const Route = createLazyFileRoute('/auth/register')({
  component: RegisterForm,
});

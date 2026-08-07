import { createLazyFileRoute } from '@tanstack/react-router';
import { LoginForm } from '@/features/auth/pages/login-form';

export const Route = createLazyFileRoute('/auth/login')({
  component: LoginForm,
});

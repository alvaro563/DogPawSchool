import { useState, type FormEvent } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { PawPrint, AlertCircle } from 'lucide-react';
import { loginSchema, type LoginInput } from '@/domain/schemas/auth-schema';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import type { ApiError } from '@/infrastructure/api/http-client';

interface FieldError {
  email?: string;
  password?: string;
}

export function LoginForm() {
  const navigate = useNavigate();
  const { login } = useAuth();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldError>({});
  const [serverError, setServerError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  function validateClient(): boolean {
    const result = loginSchema.safeParse({ email, password });
    if (result.success) {
      setFieldErrors({});
      return true;
    }
    const errors: FieldError = {};
    for (const issue of result.error.issues) {
      const field = issue.path[0] as string;
      if (field === 'email' && !errors.email) {
        errors.email = issue.message;
      }
      if (field === 'password' && !errors.password) {
        errors.password = issue.message;
      }
    }
    setFieldErrors(errors);
    return false;
  }

  function getServerErrorMessage(status: number): string {
    switch (status) {
      case 401:
        return 'Credenciales incorrectas. Verifica tu email y contraseña.';
      case 429:
        return 'Demasiados intentos. Espera un momento antes de intentarlo de nuevo.';
      case 400:
        return 'Datos inválidos. Revisa los campos.';
      default:
        return 'Error de conexión. Inténtalo de nuevo.';
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setServerError('');

    if (!validateClient()) return;

    setIsSubmitting(true);
    try {
      const input: LoginInput = { email: email.trim(), password };
      await login(input);
      const userRaw = localStorage.getItem('auth_user');
      if (userRaw) {
        const user = JSON.parse(userRaw) as { role: string };
        navigate({ to: user.role === 'ADMIN' ? '/admin' : '/calendar' });
      }
    } catch (err) {
      const apiErr = err as ApiError;
      setServerError(getServerErrorMessage(apiErr.status));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-b from-background to-muted/30 px-4 py-12 sm:px-6 lg:px-8">
      <Card className="w-full max-w-md border-border/50 shadow-lg">
        <CardHeader className="space-y-1 pb-6 pt-8 text-center">
          <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-foreground">
            <PawPrint className="h-8 w-8 text-background" strokeWidth={2.5} />
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            Dog Paw
          </CardTitle>
          <CardDescription className="text-muted-foreground">
            Bienvenid@ de nuevo al cole de perros! Te esperamos dentro.
          </CardDescription>
        </CardHeader>

        <Separator />

        <CardContent className="space-y-5 pt-6">
          <form onSubmit={handleSubmit} noValidate>
            <div className="space-y-4">
              <div className="space-y-2">
                <label
                  htmlFor="email"
                  className="pl-2.5 text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                >
                  Email
                </label>
                <Input
                  id="email"
                  type="email"
                  placeholder="tu@email.com"
                  autoComplete="email"
                  value={email}
                  onChange={(e) => {
                    setEmail(e.target.value);
                    if (fieldErrors.email) {
                      setFieldErrors((prev) => ({ ...prev, email: undefined }));
                    }
                  }}
                  disabled={isSubmitting}
                  data-invalid={!!fieldErrors.email}
                  className={fieldErrors.email ? 'border-destructive' : ''}
                />
                {fieldErrors.email && (
                  <p className="text-xs text-destructive">{fieldErrors.email}</p>
                )}
              </div>

              <div className="space-y-2">
                <label
                  htmlFor="password"
                  className="pl-2.5 text-sm font-medium leading-none"
                >
                  Contraseña
                </label>
                <Input
                  id="password"
                  type="password"
                  placeholder="••••••••"
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => {
                    setPassword(e.target.value);
                    if (fieldErrors.password) {
                      setFieldErrors((prev) => ({
                        ...prev,
                        password: undefined,
                      }));
                    }
                  }}
                  disabled={isSubmitting}
                  data-invalid={!!fieldErrors.password}
                  className={fieldErrors.password ? 'border-destructive' : ''}
                />
                {fieldErrors.password && (
                  <p className="text-xs text-destructive">
                    {fieldErrors.password}
                  </p>
                )}
              </div>
            </div>

            {serverError && (
              <div className="mt-4 flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                <span>{serverError}</span>
              </div>
            )}

            <Button
              type="submit"
              className="mt-6 w-full"
              disabled={isSubmitting}
              size="lg"
            >
              {isSubmitting ? (
                <LoadingSpinner size="sm" className="border-t-background" />
              ) : (
                'Iniciar sesión'
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

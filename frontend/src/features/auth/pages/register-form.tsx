import { useEffect, useState, type FormEvent } from 'react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { PawPrint, AlertCircle } from 'lucide-react';
import { registerSchema, type RegisterInput } from '@/domain/schemas/auth-schema';
import { AuthRepositoryImpl } from '@/infrastructure/repositories/auth-repository.impl';
import storageToken from '@/infrastructure/storage/token';
import { userStorage } from '@/infrastructure/storage/user';
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
  name?: string;
  password?: string;
}

function formatCountdown(seconds: number): string {
  if (seconds <= 0) return '0s';
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m === 0) return `${s}s`;
  return `${m}m ${s.toString().padStart(2, '0')}s`;
}

function getServerErrorMessage(status: number): string {
  switch (status) {
    case 409:
      return 'Este enlace ya fue utilizado o expiró. Solicita una nueva invitación.';
    case 400:
      return 'Datos inválidos. Revisa los campos.';
    case 404:
      return 'Enlace inválido.';
    case 429:
      return 'Demasiados intentos. Espera antes de intentarlo de nuevo.';
    default:
      return 'Error de conexión. Inténtalo de nuevo.';
  }
}

export function RegisterForm() {
  const navigate = useNavigate();
  const authRepo = new AuthRepositoryImpl();

  let searchToken: string | undefined;
  try {
    const search = useSearch({ strict: false }) as Record<string, unknown>;
    searchToken = typeof search.token === 'string' ? search.token : undefined;
  } catch {
    searchToken = undefined;
  }

  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldError>({});
  const [serverError, setServerError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [retryAfterSeconds, setRetryAfterSeconds] = useState<number | null>(null);

  useEffect(() => {
    if (retryAfterSeconds === null || retryAfterSeconds <= 0) return;
    const id = window.setInterval(() => {
      setRetryAfterSeconds((s) => (s === null ? null : s - 1));
    }, 1000);
    return () => window.clearInterval(id);
  }, [retryAfterSeconds]);

  if (!searchToken) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gradient-to-b from-background to-muted/30 px-4 py-12 sm:px-6 lg:px-8">
        <Card className="w-full max-w-md border-border/50 shadow-lg">
          <CardHeader className="space-y-1 pb-6 pt-8 text-center">
            <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-foreground">
              <PawPrint className="h-8 w-8 text-background" strokeWidth={2.5} />
            </div>
            <CardTitle className="text-2xl font-bold tracking-tight">Dog Paw</CardTitle>
            <CardDescription className="text-muted-foreground">
              Enlace inválido o no proporcionado.
            </CardDescription>
          </CardHeader>
          <Separator />
          <CardContent className="space-y-5 pt-6 text-center">
            <p className="text-sm text-muted-foreground">
              Solicita una nueva invitación a tu administrador.
            </p>
            <Button variant="outline" className="w-full" onClick={() => navigate({ to: '/auth/login' })}>
              Ir al login
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  function validateClient(): boolean {
    const result = registerSchema.safeParse({ token: searchToken, name, password });
    if (result.success) {
      setFieldErrors({});
      return true;
    }
    const errors: FieldError = {};
    for (const issue of result.error.issues) {
      const field = issue.path[0] as string;
      if (field === 'name' && !errors.name) errors.name = issue.message;
      if (field === 'password' && !errors.password) errors.password = issue.message;
    }
    setFieldErrors(errors);
    return false;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setServerError('');

    if (retryAfterSeconds !== null && retryAfterSeconds > 0) return;

    if (!validateClient()) return;

    setIsSubmitting(true);
    try {
      const input: RegisterInput = { token: searchToken!, name: name.trim(), password };
      const response = await authRepo.registerWithInvitation(input);

      storageToken.set(response.token);
      userStorage.set(response.user);
      navigate({ to: response.user.role === 'ADMIN' ? '/admin' : '/calendar' });
    } catch (err) {
      const apiErr = err as ApiError;
      if (apiErr.status === 429 && apiErr.retryAfterSeconds != null) {
        setRetryAfterSeconds(apiErr.retryAfterSeconds);
        setServerError('');
      } else {
        setServerError(getServerErrorMessage(apiErr.status));
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  const isLockedOut = retryAfterSeconds !== null && retryAfterSeconds > 0;
  const submitDisabled = isSubmitting || isLockedOut;
  const submitLabel = isLockedOut
    ? `Espera ${formatCountdown(retryAfterSeconds)}`
    : isSubmitting
      ? null
      : 'Crear cuenta';

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-b from-background to-muted/30 px-4 py-12 sm:px-6 lg:px-8">
      <Card className="w-full max-w-md border-border/50 shadow-lg">
        <CardHeader className="space-y-1 pb-6 pt-8 text-center">
          <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-foreground">
            <PawPrint className="h-8 w-8 text-background" strokeWidth={2.5} />
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">Dog Paw</CardTitle>
          <CardDescription className="text-muted-foreground">
            Crea tu cuenta para acceder al cole de perros.
          </CardDescription>
        </CardHeader>

        <Separator />

        <CardContent className="space-y-5 pt-6">
          <form onSubmit={handleSubmit} noValidate>
            <div className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="name" className="pl-2.5 text-sm font-medium leading-none">
                  Nombre
                </label>
                <Input
                  id="name"
                  type="text"
                  placeholder="Tu nombre"
                  autoComplete="name"
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value);
                    if (fieldErrors.name) setFieldErrors((prev) => ({ ...prev, name: undefined }));
                  }}
                  disabled={isSubmitting || isLockedOut}
                  data-invalid={!!fieldErrors.name}
                  className={fieldErrors.name ? 'border-destructive' : ''}
                />
                {fieldErrors.name && <p className="text-xs text-destructive">{fieldErrors.name}</p>}
              </div>

              <div className="space-y-2">
                <label htmlFor="password" className="pl-2.5 text-sm font-medium leading-none">
                  Contraseña
                </label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Mínimo 8 caracteres"
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => {
                    setPassword(e.target.value);
                    if (fieldErrors.password) setFieldErrors((prev) => ({ ...prev, password: undefined }));
                  }}
                  disabled={isSubmitting || isLockedOut}
                  data-invalid={!!fieldErrors.password}
                  className={fieldErrors.password ? 'border-destructive' : ''}
                />
                {fieldErrors.password && <p className="text-xs text-destructive">{fieldErrors.password}</p>}
              </div>
            </div>

            {serverError && (
              <div className="mt-4 flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                <span>{serverError}</span>
              </div>
            )}

            {isLockedOut && (
              <div
                className="mt-4 flex items-start gap-2 rounded-md bg-amber-100 p-3 text-sm text-amber-900 dark:bg-amber-900/30 dark:text-amber-100"
                role="status"
                aria-live="polite"
              >
                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                <span>
                  Demasiados intentos desde tu origen. Podrás intentarlo de
                  nuevo en <strong>{formatCountdown(retryAfterSeconds)}</strong>.
                </span>
              </div>
            )}

            <Button type="submit" className="mt-6 w-full" disabled={submitDisabled} size="lg">
              {isSubmitting ? (
                <LoadingSpinner size="sm" className="border-t-background" />
              ) : (
                submitLabel
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

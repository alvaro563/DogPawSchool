import { useEffect, useState, type FormEvent } from 'react';
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

function formatCountdown(seconds: number): string {
  if (seconds <= 0) return '0s';
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m === 0) return `${s}s`;
  return `${m}m ${s.toString().padStart(2, '0')}s`;
}

export function LoginForm() {
  const navigate = useNavigate();
  const { login } = useAuth();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldError>({});
  const [serverError, setServerError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 429 countdown. Null when no lockout is active. Counts down
  // once per second; submit button is disabled while positive.
  const [retryAfterSeconds, setRetryAfterSeconds] = useState<number | null>(
    null,
  );

  useEffect(() => {
    if (retryAfterSeconds === null || retryAfterSeconds <= 0) return;
    const id = window.setInterval(() => {
      setRetryAfterSeconds((s) => (s === null ? null : s - 1));
    }, 1000);
    return () => window.clearInterval(id);
  }, [retryAfterSeconds]);

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

  function getServerErrorMessage(apiErr: ApiError): string {
    // status === undefined means the fetch never reached the
    // server (network failure, CORS rejection, CSP block). The
    // http-client surfaces this as a sentinel ApiError so we can
    // distinguish it from a server-side rejection.
    if (apiErr.status === undefined) {
      return 'No se pudo conectar con el servidor. ¿Está el API en marcha?';
    }
    switch (apiErr.status) {
      case 401:
        return 'Credenciales incorrectas. Verifica tu email y contraseña.';
      case 429:
        return 'Demasiados intentos. Espera antes de intentarlo de nuevo.';
      case 400:
        return 'Datos inválidos. Revisa los campos.';
      default:
        return 'Error de conexión. Inténtalo de nuevo.';
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setServerError('');

    if (retryAfterSeconds !== null && retryAfterSeconds > 0) return;

    if (!validateClient()) return;

    setIsSubmitting(true);
    try {
      const input: LoginInput = { email: email.trim().toLowerCase(), password };
      const user = await login(input);
      navigate({ to: user.role === 'ADMIN' ? '/admin' : '/calendar' });
    } catch (err) {
      const apiErr = err as ApiError;
      if (apiErr.status === 429 && apiErr.retryAfterSeconds != null) {
        setRetryAfterSeconds(apiErr.retryAfterSeconds);
        setServerError('');
      } else {
        setServerError(getServerErrorMessage(apiErr));
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
      : 'Iniciar sesión';

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
                  disabled={isSubmitting || isLockedOut}
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
                  disabled={isSubmitting || isLockedOut}
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

            {isLockedOut && (
              <div
                className="mt-4 flex items-start gap-2 rounded-md bg-amber-100 p-3 text-sm text-amber-900 dark:bg-amber-900/30 dark:text-amber-100"
                role="status"
                aria-live="polite"
              >
                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                <span>
                  Cuenta bloqueada temporalmente por demasiados intentos.
                  Podrás intentarlo de nuevo en{' '}
                  <strong>{formatCountdown(retryAfterSeconds)}</strong>.
                </span>
              </div>
            )}

            <Button
              type="submit"
              className="mt-6 w-full"
              disabled={submitDisabled}
              size="lg"
            >
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

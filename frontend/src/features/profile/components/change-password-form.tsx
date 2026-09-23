import { useState, type FormEvent } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { KeyRound, Save, X, AlertCircle, Eye, EyeOff } from 'lucide-react';
import { useToast } from '@/features/ui/hooks/toast-context';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { AuthRepositoryImpl } from '@/infrastructure/repositories/auth-repository.impl';
import type { ApiError } from '@/infrastructure/api/http-client';
import { Button } from '@/components/ui/button';
import { LoadingSpinner } from '@/components/shared/loading-spinner';

const authRepository = new AuthRepositoryImpl();

// Password length bounds. Mirrored from the backend
// (internal/usecase/auth/change_password.go:37-42): 8 to 72 chars,
// where 72 is the bcrypt input limit. Keeping the bounds in lockstep
// lets the form reject obviously-wrong input before a roundtrip.
const MIN_PASSWORD_LEN = 8;
const MAX_PASSWORD_LEN = 72;

interface ChangePasswordFormProps {
  // Called when the user clicks "Cancelar" so the parent can hide
  // the form. The form does not manage its own visibility — that
  // belongs to ProfilePage where the toggle button lives.
  onClose: () => void;
}

// parseError translates a PATCH /auth/password ApiError into a
// user-facing Spanish string. 401 from this endpoint is a domain
// rejection (wrong old password, NOT session expired) — we read the
// wire code from the body and surface it inline without redirecting
// the user to /auth/login (the http-client is told via
// `skipAuthRedirect: true` not to).
function parseError(err: unknown): string {
  const apiErr = err as ApiError;
  const body = apiErr?.body as { error?: string; field?: string; details?: string } | null;
  switch (body?.error) {
    case 'invalid_credentials':
      return 'La contraseña actual es incorrecta.';
    case 'same_password':
      return 'La nueva contraseña no puede ser igual a la actual.';
    case 'validation':
      if (body.field === 'new_password') {
        return `La nueva contraseña debe tener entre ${MIN_PASSWORD_LEN} y ${MAX_PASSWORD_LEN} caracteres.`;
      }
      return body.details ?? 'Datos inválidos.';
    default:
      return 'No se pudo cambiar la contraseña. Inténtalo de nuevo.';
  }
}

// ChangePasswordForm renders a 3-field inline form (current / new /
// repeat) inside ProfilePage. Submission hits PATCH /auth/password;
// on success the auth context is force-cleared and the user is sent
// to /auth/login because the backend bumps the user's token_version,
// which invalidates the current JWT on the next request.
export function ChangePasswordForm({ onClose }: ChangePasswordFormProps) {
  const toast = useToast();
  const navigate = useNavigate();
  const { logout } = useAuth();

  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPasswords, setShowPasswords] = useState(false);
  const [clientError, setClientError] = useState('');

  const mutation = useMutation({
    mutationFn: () =>
      authRepository.changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      }),
    onSuccess: () => {
      // The backend has already incremented the user's token_version
      // (ChangePasswordUseCase does this). Any subsequent request
      // with the current JWT returns 401 from AuthRequired. We force
      // the redirect here instead of waiting for the next 401 — UX
      // is "you changed your password, log back in" not "your next
      // click mysteriously fails".
      toast.success(
        'Contraseña actualizada',
        'Vuelve a iniciar sesión con la nueva contraseña.',
      );
      // Fire-and-forget: the hook's logout POSTs /auth/logout
      // (server clears cookies) and clears local state. We do
      // not block the navigation on the network call because the
      // local state is cleared regardless.
      void logout();
      navigate({ to: '/auth/login' });
    },
    onError: (err: unknown) => {
      setClientError(parseError(err));
    },
  });

  function validateClient(): string {
    if (!oldPassword) return 'Introduce tu contraseña actual.';
    if (!newPassword) return 'Introduce la nueva contraseña.';
    if (newPassword.length < MIN_PASSWORD_LEN || newPassword.length > MAX_PASSWORD_LEN) {
      return `La nueva contraseña debe tener entre ${MIN_PASSWORD_LEN} y ${MAX_PASSWORD_LEN} caracteres.`;
    }
    if (newPassword === oldPassword) {
      return 'La nueva contraseña no puede ser igual a la actual.';
    }
    if (newPassword !== confirmPassword) {
      return 'La nueva contraseña y su confirmación no coinciden.';
    }
    return '';
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const v = validateClient();
    if (v) {
      setClientError(v);
      return;
    }
    setClientError('');
    mutation.mutate();
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4" noValidate>
      <div className="flex items-center gap-2 text-sm font-medium">
        <KeyRound className="h-4 w-4 text-muted-foreground" />
        Cambiar contraseña
      </div>

      <div className="space-y-3">
        <div className="space-y-1.5">
          <label className="text-xs font-medium" htmlFor="old_password">
            Contraseña actual
          </label>
          <input
            id="old_password"
            name="old_password"
            type={showPasswords ? 'text' : 'password'}
            autoComplete="current-password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            disabled={mutation.isPending}
            className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
            required
          />
        </div>

        <div className="space-y-1.5">
          <label className="text-xs font-medium" htmlFor="new_password">
            Nueva contraseña
          </label>
          <input
            id="new_password"
            name="new_password"
            type={showPasswords ? 'text' : 'password'}
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            disabled={mutation.isPending}
            minLength={MIN_PASSWORD_LEN}
            maxLength={MAX_PASSWORD_LEN}
            className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
            required
          />
          <p className="text-[10px] text-muted-foreground">
            Entre {MIN_PASSWORD_LEN} y {MAX_PASSWORD_LEN} caracteres. No puede ser igual a la actual.
          </p>
        </div>

        <div className="space-y-1.5">
          <label className="text-xs font-medium" htmlFor="confirm_password">
            Repetir nueva contraseña
          </label>
          <input
            id="confirm_password"
            name="confirm_password"
            type={showPasswords ? 'text' : 'password'}
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            disabled={mutation.isPending}
            minLength={MIN_PASSWORD_LEN}
            maxLength={MAX_PASSWORD_LEN}
            className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
            required
          />
        </div>

        <label className="flex cursor-pointer items-center gap-2 text-xs text-muted-foreground">
          <input
            type="checkbox"
            checked={showPasswords}
            onChange={(e) => setShowPasswords(e.target.checked)}
            className="h-3.5 w-3.5"
          />
          Mostrar contraseñas
          {showPasswords ? <Eye className="h-3 w-3" /> : <EyeOff className="h-3 w-3" />}
        </label>
      </div>

      {clientError && (
        <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
          {clientError}
        </div>
      )}

      <div className="flex items-center gap-2">
        <Button type="submit" size="sm" disabled={mutation.isPending}>
          {mutation.isPending ? (
            <LoadingSpinner size="sm" className="border-t-background" />
          ) : (
            <>
              <Save className="h-3.5 w-3.5" />
              Guardar
            </>
          )}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => {
            // Reset before closing so a re-open shows a clean form.
            setOldPassword('');
            setNewPassword('');
            setConfirmPassword('');
            setShowPasswords(false);
            setClientError('');
            mutation.reset();
            onClose();
          }}
          disabled={mutation.isPending}
        >
          <X className="h-3.5 w-3.5" />
          Cancelar
        </Button>
      </div>
    </form>
  );
}

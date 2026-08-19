import { useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, CheckCircle2, Copy } from 'lucide-react';
import { createInvitation } from '@/infrastructure/repositories/invitation-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { useToast } from '@/features/ui/hooks/toast-context';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

interface RegisterClientModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RegisterClientModal({ open, onOpenChange }: RegisterClientModalProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('REGULAR');
  const [error, setError] = useState('');
  const [inviteToken, setInviteToken] = useState('');
  const [copied, setCopied] = useState(false);

  const mutation = useMutation({
    mutationFn: () => createInvitation(email, role),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'], refetchType: 'all' });
      setInviteToken(data.token);
      setError('');
      toast.success(
        'Invitación creada',
        `Enlace de alta enviado a ${email}. El cliente puede registrarse con él.`,
      );
    },
    onError: (err: unknown) => {
      setError(parseError(err, 'Error al crear la invitación.'));
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!email) {
      setError('Introduce un email');
      return;
    }
    setInviteToken('');
    setError('');
    mutation.mutate();
  }

  function handleCopy() {
    navigator.clipboard.writeText(inviteToken);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  function handleClose() {
    setEmail('');
    setInviteToken('');
    setError('');
    onOpenChange(false);
  }

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Dar de alta cliente</SheetTitle>
        </SheetHeader>

        {inviteToken ? (
          <div className="mt-6 flex flex-col items-center gap-4 px-4 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-100 dark:bg-emerald-900/30">
              <CheckCircle2 className="h-7 w-7 text-emerald-600" />
            </div>
            <div>
              <p className="font-semibold">Invitación creada</p>
              <p className="text-sm text-muted-foreground">Comparte este código con el cliente</p>
            </div>
            <div className="flex w-full items-center gap-2 rounded-lg border border-border bg-muted/50 p-3">
              <code className="flex-1 break-all text-xs font-mono">{inviteToken}</code>
              <button onClick={handleCopy} className="shrink-0 rounded p-1 transition-colors hover:bg-muted">
                {copied ? <CheckCircle2 className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
              </button>
            </div>
            <Button variant="outline" className="mt-2 w-full" onClick={handleClose}>
              Cerrar
            </Button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="mt-4 space-y-3 px-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Email</label>
              <input
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
                type="email"
                placeholder="cliente@ejemplo.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Rol</label>
              <select
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
                value={role}
                onChange={(e) => setRole(e.target.value)}
              >
                <option value="REGULAR">Cliente</option>
                <option value="ADMIN">Administrador</option>
            </select>
            </div>

            {error && (
              <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
                <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
                {error}
              </div>
            )}

            <Button type="submit" className="w-full" disabled={mutation.isPending}>
              {mutation.isPending ? <LoadingSpinner size="sm" className="border-t-background" /> : 'Crear invitación'}
            </Button>
          </form>
        )}
      </SheetContent>
    </Sheet>
  );
}

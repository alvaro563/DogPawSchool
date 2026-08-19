import { useState, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle } from 'lucide-react';
import { fetchAllUsers } from '@/infrastructure/repositories/user-repository.impl';
import { createPass } from '@/infrastructure/repositories/pass-repository.impl';
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

interface AssignPassModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AssignPassModal({ open, onOpenChange }: AssignPassModalProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [userId, setUserId] = useState<number | null>(null);
  const [numSessions, setNumSessions] = useState(10);
  const [price, setPrice] = useState(40);
  const [passType, setPassType] = useState('GENERICO');
  const [expiresAt, setExpiresAt] = useState('');
  const [error, setError] = useState('');

  const { data: users = [], isLoading: usersLoading } = useQuery({
    queryKey: ['all-users'],
    queryFn: fetchAllUsers,
    enabled: open,
  });

  const mutation = useMutation({
    mutationFn: () =>
      createPass(userId!, {
        num_of_sessions: numSessions,
        price: price * 100,
        pass_type: passType,
        expires_at: expiresAt ? new Date(`${expiresAt}T23:59:59`).toISOString() : undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'], refetchType: 'all' });
      setUserId(null);
      setError('');

      const owner = users.find((u) => u.id === userId);
      const ownerName = owner?.name ?? 'el cliente';
      const passTypeLabel = passType === 'GENERICO' ? 'genérico' : 'específico';
      toast.success(
        'Bono asignado',
        `${numSessions} sesiones (${passTypeLabel}) para ${ownerName}.`,
      );

      onOpenChange(false);
    },
    onError: (err: unknown) => {
      setError(parseError(err, 'Error al crear el bono.'));
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!userId) {
      setError('Selecciona un cliente');
      return;
    }
    setError('');
    mutation.mutate();
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Asignar bono</SheetTitle>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="mt-4 space-y-3 px-4">
          {usersLoading ? (
            <LoadingSpinner />
          ) : (
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Cliente</label>
              <select className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={userId ?? ''} onChange={(e) => setUserId(e.target.value ? +e.target.value : null)} required>
                <option value="">Seleccionar cliente</option>
                {users.filter((u) => u.is_active).map((u) => (
                  <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                ))}
              </select>
            </div>
          )}
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Nº de sesiones</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="number" value={numSessions} min={1} onChange={(e) => setNumSessions(+e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Precio (€)</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="number" step={1} value={price} min={0} onChange={(e) => setPrice(+e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Tipo de bono</label>
            <select className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={passType} onChange={(e) => setPassType(e.target.value)}>
              <option value="GENERICO">Genérico</option>
              <option value="ESPECIFICO">Específico</option>
            </select>
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Fecha de expiración (opcional)</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="date" value={expiresAt} onChange={(e) => setExpiresAt(e.target.value)} />
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
              <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
              {error}
            </div>
          )}

          <Button type="submit" className="w-full" disabled={mutation.isPending || !userId}>
            {mutation.isPending ? <LoadingSpinner size="sm" className="border-t-background" /> : 'Crear bono'}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}

import { useState, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle } from 'lucide-react';
import { fetchAllUsers } from '@/infrastructure/repositories/user-repository.impl';
import { registerDog } from '@/infrastructure/repositories/dog-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

interface RegisterDogModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RegisterDogModal({ open, onOpenChange }: RegisterDogModalProps) {
  const queryClient = useQueryClient();
  const [ownerId, setOwnerId] = useState<number | null>(null);
  const [name, setName] = useState('');
  const [breed, setBreed] = useState('');
  const [ageMonths, setAgeMonths] = useState(24);
  const [sex, setSex] = useState('FEMALE');
  const [weightKg, setWeightKg] = useState(20);
  const [passport, setPassport] = useState('');
  const [error, setError] = useState('');

  const { data: users = [], isLoading: usersLoading } = useQuery({
    queryKey: ['all-users'],
    queryFn: fetchAllUsers,
    enabled: open,
  });

  const mutation = useMutation({
    mutationFn: () =>
      registerDog({
        name,
        breed,
        age_in_months: ageMonths,
        sex,
        weight_kg: weightKg,
        passport,
        user_id: ownerId!,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'], refetchType: 'all' });
      setName('');
      setBreed('');
      setPassport('');
      setError('');
      onOpenChange(false);
    },
    onError: (err: unknown) => {
      setError(parseError(err, 'Error al registrar el perro.'));
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!ownerId || !name || !breed || !passport) {
      setError('Completa todos los campos obligatorios');
      return;
    }
    setError('');
    mutation.mutate();
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Registrar perro</SheetTitle>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="mt-4 space-y-3 px-4">
          {usersLoading ? (
            <LoadingSpinner />
          ) : (
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Dueño</label>
              <select className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={ownerId ?? ''} onChange={(e) => setOwnerId(e.target.value ? +e.target.value : null)} required>
                <option value="">Seleccionar dueño</option>
                {users.filter((u) => u.is_active).map((u) => (
                  <option key={u.id} value={u.id}>{u.name}</option>
                ))}
              </select>
            </div>
          )}
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Nombre</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Raza</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={breed} onChange={(e) => setBreed(e.target.value)} required />
          </div>
          <div className="flex gap-2">
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Edad (meses)</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="number" value={ageMonths} min={1} onChange={(e) => setAgeMonths(+e.target.value)} />
            </div>
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Sexo</label>
              <select className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={sex} onChange={(e) => setSex(e.target.value)}>
                <option value="FEMALE">Hembra</option>
                <option value="MALE">Macho</option>
              </select>
            </div>
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Peso (kg)</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="number" value={weightKg} min={1} step={0.1} onChange={(e) => setWeightKg(+e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Pasaporte</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" placeholder="ES-12345" value={passport} onChange={(e) => setPassport(e.target.value)} required />
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
              <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
              {error}
            </div>
          )}

          <Button type="submit" className="w-full" disabled={mutation.isPending || !ownerId}>
            {mutation.isPending ? <LoadingSpinner size="sm" className="border-t-background" /> : 'Registrar perro'}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}

import { useState, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle } from 'lucide-react';
import apiClient from '@/infrastructure/api/http-client';
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

const ACTIVITY_TYPES = [
  { value: 'SOCIALIZATION_GROUP', label: 'Grupo de socialización' },
  { value: 'ROUTE', label: 'Ruta' },
  { value: 'INDIVIDUAL_CLASS', label: 'Clase individual' },
  { value: 'EXTRA', label: 'Evento extra' },
];

interface CreateActivityModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateActivityModal({ open, onOpenChange }: CreateActivityModalProps) {
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [activityType, setActivityType] = useState('SOCIALIZATION_GROUP');
  const [maxCapacity, setMaxCapacity] = useState(8);
  const [durationInHours, setDurationInHours] = useState(2);
  const [date, setDate] = useState('');
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: () =>
      apiClient.post('/activities', {
        name,
        description,
        location,
        activity_type: activityType,
        max_capacity: maxCapacity,
        duration_in_hours: durationInHours,
        date: new Date(date).toISOString(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'], refetchType: 'all' });
      setName('');
      setDescription('');
      setLocation('');
      setDate('');
      setError('');
      onOpenChange(false);
    },
    onError: (err: unknown) => {
      setError(parseError(err, 'Error al crear la actividad.'));
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!name || !location || !date) {
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
          <SheetTitle>Crear nueva actividad</SheetTitle>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="mt-4 space-y-3 px-4">
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Nombre</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Descripción (opcional)</label>
            <textarea className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={description} onChange={(e) => setDescription(e.target.value)} rows={2} />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Ubicación</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={location} onChange={(e) => setLocation(e.target.value)} required />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Tipo de actividad</label>
            <select className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={activityType} onChange={(e) => setActivityType(e.target.value)}>
              {ACTIVITY_TYPES.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
            </select>
          </div>
          <div className="flex gap-2">
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Plazas</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="number" value={maxCapacity} min={1} onChange={(e) => setMaxCapacity(+e.target.value)} />
            </div>
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Duración (horas)</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="number" value={durationInHours} min={1} onChange={(e) => setDurationInHours(+e.target.value)} />
            </div>
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium">Fecha y hora</label>
            <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="datetime-local" value={date} onChange={(e) => setDate(e.target.value)} required />
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
              <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
              {error}
            </div>
          )}

          <Button type="submit" className="w-full" disabled={mutation.isPending}>
            {mutation.isPending ? <LoadingSpinner size="sm" className="border-t-background" /> : 'Crear actividad'}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}

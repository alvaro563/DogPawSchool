import { useState, useRef, useEffect, useCallback, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, ChevronUp, ChevronDown } from 'lucide-react';
import apiClient from '@/infrastructure/api/http-client';
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

const ACTIVITY_TYPES = [
  { value: 'SOCIALIZATION_GROUP', label: 'Grupo de socialización' },
  { value: 'ROUTE', label: 'Ruta' },
  { value: 'INDIVIDUAL_CLASS', label: 'Clase individual' },
  { value: 'EXTRA', label: 'Evento extra' },
];

interface TimePickerProps {
  value: string;
  onChange: (v: string) => void;
}

function TimePicker({ value, onChange }: TimePickerProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const parts = value.split(':');
  const hours = parseInt(parts[0], 10) || 0;
  const minutes = parseInt(parts[1], 10) || 0;

  const build = useCallback(
    (h: number, m: number) =>
      `${String(((h % 24) + 24) % 24).padStart(2, '0')}:${String(((m % 60) + 60) % 60).padStart(2, '0')}`,
    [],
  );

  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [open]);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex h-9 w-full items-center justify-between rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
      >
        {value}
      </button>
      {open && (
        <div className="absolute z-50 mt-1 rounded-lg border border-border bg-popover p-3 shadow-lg">
          <div className="flex items-center gap-4">
            <div className="flex flex-col items-center gap-1">
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => onChange(build(hours + 1, minutes))}>
                <ChevronUp className="h-4 w-4" />
              </Button>
              <span className="w-10 text-center text-sm font-medium tabular-nums">
                {String(hours).padStart(2, '0')}
              </span>
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => onChange(build(hours - 1, minutes))}>
                <ChevronDown className="h-4 w-4" />
              </Button>
            </div>
            <span className="self-center text-lg font-bold">:</span>
            <div className="flex flex-col items-center gap-1">
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => onChange(build(hours, minutes + 1))}>
                <ChevronUp className="h-4 w-4" />
              </Button>
              <span className="w-10 text-center text-sm font-medium tabular-nums">
                {String(minutes).padStart(2, '0')}
              </span>
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => onChange(build(hours, minutes - 1))}>
                <ChevronDown className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface CreateActivityModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateActivityModal({ open, onOpenChange }: CreateActivityModalProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [activityType, setActivityType] = useState('SOCIALIZATION_GROUP');
  const [maxCapacity, setMaxCapacity] = useState(8);
  const [durationInHours, setDurationInHours] = useState(2);
  const [datePart, setDatePart] = useState('');
  const [timePart, setTimePart] = useState('10:00');
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
        date: new Date(`${datePart}T${timePart}:00`).toISOString(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'], refetchType: 'all' });
      setName('');
      setDescription('');
      setLocation('');
      setDatePart('');
      setTimePart('10:00');
      setError('');

      const activityDate = new Date(`${datePart}T${timePart}:00`);
      const formattedDate = activityDate.toLocaleDateString('es-ES', {
        weekday: 'long',
        day: 'numeric',
        month: 'long',
      });
      toast.success(
        'Actividad creada',
        `${name} programada para ${formattedDate}. Ya aparece en el calendario.`,
      );

      onOpenChange(false);
    },
    onError: (err: unknown) => {
      setError(parseError(err, 'Error al crear la actividad.'));
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!name || !location || !datePart || !timePart) {
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
          <div className="flex gap-2">
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Fecha</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="date" value={datePart} onChange={(e) => setDatePart(e.target.value)} required />
            </div>
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Hora</label>
              <TimePicker value={timePart} onChange={setTimePart} />
            </div>
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

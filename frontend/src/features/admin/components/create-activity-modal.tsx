import { useState, useRef, useEffect, useCallback, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, ChevronUp, ChevronDown, Clock } from 'lucide-react';
import apiClient from '@/infrastructure/api/http-client';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { useToast } from '@/features/ui/hooks/toast-context';
import { fetchAllActiveDogs } from '@/infrastructure/repositories/dog-repository.impl';
import { updateActivity } from '@/infrastructure/repositories/activity-repository.impl';
import type { Activity } from '@/domain/entities/activity';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as { body?: { error?: string; field?: string; details?: string } };
  const b = apiErr.body;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

function todayLocalDate(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function openDatePicker(e: React.MouseEvent<HTMLInputElement>) {
  const el = e.currentTarget;
  if (typeof el.showPicker === 'function') {
    try {
      el.showPicker();
    } catch {
      // showPicker can throw if the input is disabled or already open;
      // fall back to the native focus+click behaviour.
    }
  }
}

const ACTIVITY_TYPES = [
  { value: 'SOCIALIZATION_GROUP', label: 'Grupo de socialización' },
  { value: 'ROUTE', label: 'Ruta' },
  { value: 'INDIVIDUAL_CLASS', label: 'Clase individual' },
  { value: 'EXTRA', label: 'Evento extra' },
];

// Recurring creates materialize one activity row per session; the
// backend caps a single batch at one year of weekly occurrences.
const MAX_SESSIONS = 52;

type RecurrenceResult = { dates: Date[] } | { error: string };

// sessionInstant returns the local Date of the weekly occurrence
// `weeksAhead` weeks after the first session. The +7 lands on the
// day component of a local (calendar) date, so the same wall-clock
// time is kept across DST transitions — unlike adding 7*24h.
function sessionInstant(datePart: string, timePart: string, weeksAhead: number): Date {
  const [y, m, d] = datePart.split('-').map(Number);
  const [hh, mm] = timePart.split(':').map(Number);
  return new Date(y, m - 1, d + 7 * weeksAhead, hh, mm, 0, 0);
}

// computeRecurringDates turns the recurrence settings into the list
// of session instants (one per week, `weeks` total), or an
// admin-facing error message.
function computeRecurringDates(
  datePart: string,
  timePart: string,
  weeks: number,
): RecurrenceResult {
  if (!datePart || !timePart) {
    return { error: 'Indica fecha y hora de la primera sesión.' };
  }
  if (!Number.isFinite(weeks) || weeks < 2) return { error: 'La repetición necesita al menos 2 sesiones.' };
  if (weeks > MAX_SESSIONS) return { error: `Máximo ${MAX_SESSIONS} sesiones (un año).` };
  return { dates: Array.from({ length: weeks }, (_, i) => sessionInstant(datePart, timePart, i)) };
}

// formatSessionDays renders "28 sep · 5 oct · 12 oct · … · 16 nov".
function formatSessionDays(dates: Date[]): string {
  const fmt = (d: Date) => d.toLocaleDateString('es-ES', { day: 'numeric', month: 'short' });
  const head = dates.slice(0, 3).map(fmt);
  if (dates.length <= 3) return head.join(' · ');
  return [...head, '…', fmt(dates[dates.length - 1])].join(' · ');
}

interface TimePickerProps {
  value: string;
  onChange: (v: string) => void;
}

// parseTime normalises free-typed input to canonical "HH:MM".
// Accepts "14:30", "9:30", "1430" and "930"; returns null for
// anything unparseable or outside 00:00–23:59.
function parseTime(raw: string): string | null {
  const s = raw.trim();
  let hh: number;
  let mm: number;
  const withColon = /^(\d{1,2}):(\d{1,2})$/.exec(s);
  if (withColon) {
    hh = Number(withColon[1]);
    mm = Number(withColon[2]);
  } else if (/^\d{3,4}$/.test(s)) {
    const padded = s.padStart(4, '0');
    hh = Number(padded.slice(0, 2));
    mm = Number(padded.slice(2));
  } else {
    return null;
  }
  if (hh > 23 || mm > 59) return null;
  return `${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}`;
}

// TimePicker: a text input (numeric keypad on mobile, free typing
// everywhere) combined with the original chevron stepper popover.
// The typed text lives in local `draft`; the parent's canonical
// "HH:MM" only changes on blur/Enter when the draft parses, or via
// the chevrons — so a half-typed "143" is never fed to the submit.
// Unparseable text snaps back to the last valid value on blur.
function TimePicker({ value, onChange }: TimePickerProps) {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState(value);
  const ref = useRef<HTMLDivElement>(null);

  // Sync the visible text whenever the parent commits a new value
  // (chevrons, edit-mode hydration, snap-back).
  useEffect(() => {
    setDraft(value);
  }, [value]);

  const parts = (parseTime(draft) ?? value).split(':');
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

  function commitDraft() {
    const parsed = parseTime(draft);
    if (parsed) {
      setDraft(parsed);
      if (parsed !== value) onChange(parsed);
    } else {
      // Unparseable or empty: snap back to the last valid time.
      setDraft(value);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key !== 'Enter') return;
    const parsed = parseTime(draft);
    if (parsed === null) {
      e.preventDefault();
      setDraft(value);
      return;
    }
    if (parsed !== value) {
      // Commit first and swallow this Enter so the form never
      // submits with a stale timePart; the next Enter (state now
      // already valid) goes through.
      e.preventDefault();
      setDraft(parsed);
      onChange(parsed);
    }
    // parsed === value → let the native form submit proceed.
  }

  function step(deltaH: number, deltaM: number) {
    const [h, m] = (parseTime(draft) ?? value).split(':').map(Number);
    const next = build(h + deltaH, m + deltaM);
    setDraft(next);
    onChange(next);
  }

  return (
    <div ref={ref} className="relative flex gap-1">
      <input
        type="text"
        inputMode="numeric"
        maxLength={5}
        placeholder="HH:MM"
        aria-label="Hora"
        value={draft}
        onChange={(e) => setDraft(e.target.value.replace(/[^0-9:]/g, '').slice(0, 5))}
        onBlur={commitDraft}
        onKeyDown={handleKeyDown}
        className="h-9 min-w-0 flex-1 rounded-lg border border-input bg-transparent px-2.5 text-sm"
      />
      <button
        type="button"
        aria-label="Ajustar hora con botones"
        title="Ajustar hora con botones"
        onClick={() => setOpen(!open)}
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-input bg-transparent"
      >
        <Clock className="h-4 w-4" />
      </button>
      {open && (
        <div className="absolute right-0 top-full z-50 mt-1 rounded-lg border border-border bg-popover p-3 shadow-lg">
          <div className="flex items-center gap-4">
            <div className="flex flex-col items-center gap-1">
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => step(1, 0)}>
                <ChevronUp className="h-4 w-4" />
              </Button>
              <span className="w-10 text-center text-sm font-medium tabular-nums">
                {String(hours).padStart(2, '0')}
              </span>
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => step(-1, 0)}>
                <ChevronDown className="h-4 w-4" />
              </Button>
            </div>
            <span className="self-center text-lg font-bold">:</span>
            <div className="flex flex-col items-center gap-1">
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => step(0, 1)}>
                <ChevronUp className="h-4 w-4" />
              </Button>
              <span className="w-10 text-center text-sm font-medium tabular-nums">
                {String(minutes).padStart(2, '0')}
              </span>
              <Button type="button" variant="ghost" size="icon-xs" onClick={() => step(0, -1)}>
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
  // Activity being edited. When set, the modal switches to edit mode:
  // fields hydrate from it, submit PATCHes instead of POST, and the
  // target-dog field becomes read-only (the backend's patch has no
  // dog_id — the dog of an individual class is fixed at creation).
  activity?: Activity | null;
  // Floor for max_capacity: slots already held (confirmed + pending)
  // when editing, so a patch can never push available_spots negative.
  // Defaults to 1, the backend floor, in create mode.
  minCapacity?: number;
}

export function CreateActivityModal({ open, onOpenChange, activity, minCapacity = 1 }: CreateActivityModalProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const isEdit = Boolean(activity);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [location, setLocation] = useState('');
  const [activityType, setActivityType] = useState('SOCIALIZATION_GROUP');
  const [maxCapacity, setMaxCapacity] = useState(8);
  const [durationInHours, setDurationInHours] = useState(2);
  const [datePart, setDatePart] = useState(() => todayLocalDate());
  const [timePart, setTimePart] = useState('10:00');
  const [dogId, setDogId] = useState<number | null>(null);
  const [sizeTarget, setSizeTarget] = useState<'' | 'MINI' | 'MEDIUM' | 'LARGE'>('');
  // Recurrence (create mode only): weekly repetition of the first
  // session, materialized as N independent activities in one POST.
  const [repeat, setRepeat] = useState(false);
  const [repeatWeeks, setRepeatWeeks] = useState(4);
  const [error, setError] = useState('');

  // Live preview/validation of the recurrence settings. Only the
  // controls feed it; in edit mode there is no recurrence block.
  const recurrence = !isEdit && repeat
    ? computeRecurringDates(datePart, timePart, repeatWeeks)
    : null;

  // Fetch active dogs only when the modal is open AND the admin picked
  // INDIVIDUAL_CLASS (other types don't need a target dog). Skipped in
  // edit mode: the dog is fixed at creation and the select is replaced
  // by a read-only note.
  const { data: dogs = [], isLoading: dogsLoading } = useQuery({
    queryKey: ['dogs', 'active'],
    queryFn: fetchAllActiveDogs,
    enabled: open && activityType === 'INDIVIDUAL_CLASS' && !isEdit,
  });

  // Reset the target dog when the type switches away from individual so
  // we never send a stale dog_id on the next submit.
  useEffect(() => {
    if (activityType !== 'INDIVIDUAL_CLASS') setDogId(null);
  }, [activityType]);

  // Reset the size target when the type switches to one that does not
  // accept it (INDIVIDUAL_CLASS / EXTRA). Mirrors the domain's
  // ErrSizeTargetNotApplicable guard.
  useEffect(() => {
    if (activityType !== 'SOCIALIZATION_GROUP' && activityType !== 'ROUTE') {
      setSizeTarget('');
    }
  }, [activityType]);

  // Individual classes always target one dog, so capacity is forced to 1.
  // When the type switches back to a group/route/extra, restore the
  // default of 8 so the field is usable again.
  //
  // prevTypeRef makes this fire only on a genuine type CHANGE. Without
  // it the effect would also run right after the edit-mode hydration
  // below (and on mount) and clobber the hydrated capacity with 8.
  const prevTypeRef = useRef(activityType);
  useEffect(() => {
    if (prevTypeRef.current === activityType) return;
    prevTypeRef.current = activityType;
    if (activityType === 'INDIVIDUAL_CLASS') {
      setMaxCapacity(1);
    } else {
      setMaxCapacity(8);
    }
  }, [activityType]);

  // Hydrate the form when the sheet opens. Edit mode copies the given
  // activity (date split into local date/time parts, matching how the
  // submit rebuilds the ISO instant); create mode resets to defaults
  // so values from a previous edit or create never leak in. The
  // hydrated/assumed type is also written into prevTypeRef so the
  // capacity effect above stays quiet while it is applied.
  useEffect(() => {
    if (!open) return;
    setError('');
    if (activity) {
      setName(activity.name);
      setDescription(activity.description ?? '');
      setLocation(activity.location);
      setActivityType(activity.activity_type);
      setMaxCapacity(activity.max_capacity);
      setDurationInHours(activity.duration_in_hours);
      setDogId(activity.dog_id ?? null);
      // UNKNOWN is part of the SizeBracket union but is never a
      // stored value (DB CHECK only allows MINI/MEDIUM/LARGE or NULL);
      // map it to "no target" so the select state stays a valid option.
      setSizeTarget(activity.size_target === 'UNKNOWN' || !activity.size_target ? '' : activity.size_target);
      const d = new Date(activity.date);
      setDatePart(
        `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`,
      );
      setTimePart(
        `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`,
      );
      prevTypeRef.current = activity.activity_type;
    } else {
      setName('');
      setDescription('');
      setLocation('');
      setActivityType('SOCIALIZATION_GROUP');
      setMaxCapacity(8);
      setDurationInHours(2);
      setDatePart(todayLocalDate());
      setTimePart('10:00');
      setDogId(null);
      setSizeTarget('');
      setRepeat(false);
      setRepeatWeeks(4);
      prevTypeRef.current = 'SOCIALIZATION_GROUP';
    }
  }, [open, activity]);

  const mutation = useMutation({
    // `dates` is the recurrence vector (null for a plain create or
    // any edit). Each entry becomes one session row server-side.
    mutationFn: (dates: Date[] | null) => {
      const dateISO = new Date(`${datePart}T${timePart}:00`).toISOString();
      if (activity) {
        const payload: {
          name: string;
          description: string;
          location: string;
          activity_type: string;
          max_capacity: number;
          duration_in_hours: number;
          date: string;
          size_target?: string;
        } = {
          name,
          description,
          location,
          activity_type: activityType,
          max_capacity: maxCapacity,
          duration_in_hours: durationInHours,
          date: dateISO,
        };
        // size_target triple-state: only send it for the types that
        // accept it ("" = clear). For other types we omit it and the
        // backend auto-clears on the type change itself. dog_id is
        // never sent: modifyActivityRequest has no such field.
        if (activityType === 'SOCIALIZATION_GROUP' || activityType === 'ROUTE') {
          payload.size_target = sizeTarget;
        }
        return updateActivity(activity.id, payload);
      }
      const payload: Record<string, unknown> = {
        name,
        description,
        location,
        activity_type: activityType,
        max_capacity: maxCapacity,
        duration_in_hours: durationInHours,
        date: dateISO,
        dog_id: activityType === 'INDIVIDUAL_CLASS' ? dogId : null,
        size_target:
          activityType === 'SOCIALIZATION_GROUP' || activityType === 'ROUTE'
            ? sizeTarget || null
            : null,
      };
      if (dates && dates.length > 0) {
        payload.dates = dates.map((d) => d.toISOString());
      }
      return apiClient.post('/activities', payload);
    },
    onSuccess: (_data, variables) => {
      // ['activities'] is the root prefix: refreshes the open list,
      // the completed list AND the calendar in one call. Today-classes
      // uses its own key, and admin-dashboard aggregates everything.
      // (Create used to invalidate only admin-dashboard, leaving the
      // activity lists stale until a manual refetch.)
      queryClient.invalidateQueries({ queryKey: ['activities'] });
      queryClient.invalidateQueries({ queryKey: ['today-classes'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard'], refetchType: 'all' });

      if (activity) {
        // The detail page consumes the roster, which embeds the
        // activity DTO — refresh it so header fields update too.
        queryClient.invalidateQueries({ queryKey: ['activity-roster', activity.id] });
        toast.success('Actividad actualizada', `${name} se ha guardado correctamente.`);
        setError('');
        onOpenChange(false);
        return;
      }

      setName('');
      setDescription('');
      setLocation('');
      setDatePart(todayLocalDate());
      setTimePart('10:00');
      setDogId(null);
      setSizeTarget('');
      setRepeat(false);
      setRepeatWeeks(4);
      setError('');

      // Batch creates answer with { id, ids }; single creates only
      // with { id }. variables carries the recurrence vector used at
      // mutate() time, so it can count sessions too.
      const ids = (_data as { ids?: number[] } | null)?.ids;
      const count = ids?.length ?? variables?.length ?? 0;
      if (count > 1) {
        toast.success(
          'Actividades creadas',
          `${count} sesiones semanales de «${name}» creadas — ya aparecen en el calendario.`,
        );
        onOpenChange(false);
        return;
      }

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
      setError(parseError(err, activity ? 'Error al guardar los cambios.' : 'Error al crear la actividad.'));
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!name || !location || !datePart || !timePart) {
      setError('Completa todos los campos obligatorios');
      return;
    }
    if (activityType === 'INDIVIDUAL_CLASS' && dogId === null) {
      setError('Selecciona el perro para la clase individual.');
      return;
    }
    if (maxCapacity < minCapacity) {
      setError(
        minCapacity > 1
          ? `Las plazas no pueden bajar de ${minCapacity}: ya hay ${minCapacity} plazas ocupadas.`
          : 'Las plazas deben ser al menos 1.',
      );
      return;
    }
    let dates: Date[] | null = null;
    if (!isEdit && repeat) {
      const r = computeRecurringDates(datePart, timePart, repeatWeeks);
      if ('error' in r) {
        setError(r.error);
        return;
      }
      dates = r.dates;
    }
    setError('');
    mutation.mutate(dates);
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{activity ? 'Editar actividad' : 'Crear nueva actividad'}</SheetTitle>
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
              {ACTIVITY_TYPES.map((t) => (
                <option
                  key={t.value}
                  value={t.value}
                  // Editing: cannot TURN a dog-less activity into an
                  // individual class — ActivityPatch has no dog_id and
                  // the DB CHECK activities_individual_requires_dog
                  // would reject the update.
                  disabled={isEdit && t.value === 'INDIVIDUAL_CLASS' && !activity?.dog_id}
                >
                  {t.label}
                </option>
              ))}
            </select>
          </div>
          {activityType === 'INDIVIDUAL_CLASS' && isEdit && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Perro</label>
              <p className="rounded-lg border border-dashed border-input px-2.5 py-2 text-sm text-muted-foreground">
                Asignado al crear la actividad — no se puede cambiar.
              </p>
            </div>
          )}
          {activityType === 'INDIVIDUAL_CLASS' && !isEdit && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Perro</label>
              <select
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
                value={dogId ?? ''}
                onChange={(e) => setDogId(e.target.value ? +e.target.value : null)}
                disabled={dogsLoading}
                required
              >
                <option value="">
                  {dogsLoading ? 'Cargando perros…' : 'Seleccionar perro…'}
                </option>
                {!dogsLoading && dogs.length === 0 && (
                  <option value="" disabled>No hay perros activos</option>
                )}
                {dogs.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.owner_name ? `${d.name} — ${d.owner_name}` : d.name}
                  </option>
                ))}
              </select>
            </div>
          )}
          <div className="flex gap-2">
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Plazas</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm disabled:opacity-60" type="number" value={maxCapacity} min={minCapacity} disabled={activityType === 'INDIVIDUAL_CLASS'} onChange={(e) => setMaxCapacity(+e.target.value)} />
            </div>
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Duración (horas)</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="number" value={durationInHours} min={1} onChange={(e) => setDurationInHours(+e.target.value)} />
            </div>
          </div>
          {(activityType === 'SOCIALIZATION_GROUP' || activityType === 'ROUTE') && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Tamaño objetivo</label>
              <select
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
                value={sizeTarget}
                onChange={(e) => setSizeTarget(e.target.value as '' | 'MINI' | 'MEDIUM' | 'LARGE')}
              >
                <option value="">Todos los tamaños</option>
                <option value="MINI">Mini (≤ 5 kg)</option>
                <option value="MEDIUM">Mediano (5–20 kg)</option>
                <option value="LARGE">Grande (&gt; 20 kg)</option>
              </select>
            </div>
          )}
          <div className="flex gap-2">
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Fecha</label>
              {/* min only in create mode: an open activity can be in the
                  past and must stay editable (the date itself may stay
                  unchanged while other fields are patched). */}
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="date" value={datePart} min={isEdit ? undefined : todayLocalDate()} onClick={openDatePicker} onChange={(e) => setDatePart(e.target.value)} required />
            </div>
            <div className="w-1/2 space-y-1.5">
              <label className="text-xs font-medium">Hora</label>
              <TimePicker value={timePart} onChange={setTimePart} />
            </div>
          </div>

          {!isEdit && (
            <div className="space-y-2 rounded-lg border border-border p-3">
              <label className="flex cursor-pointer items-center gap-2 text-xs font-medium">
                <input
                  type="checkbox"
                  checked={repeat}
                  onChange={(e) => setRepeat(e.target.checked)}
                  className="h-4 w-4 rounded border-input"
                />
                Repetir semanalmente
              </label>
              {repeat && (
                <>
                  <label className="flex w-fit items-center gap-1.5 text-xs">
                    Durante
                    <input
                      type="number"
                      min={2}
                      max={MAX_SESSIONS}
                      value={repeatWeeks}
                      onChange={(e) => setRepeatWeeks(+e.target.value)}
                      className="w-16 rounded-lg border border-input bg-transparent px-2 py-1 text-sm"
                      aria-label="Número de semanas"
                    />
                    semanas
                  </label>
                  <p className={`text-xs ${ recurrence && 'error' in recurrence ? 'text-destructive' : 'text-muted-foreground'}`}>
                    {recurrence && 'error' in recurrence
                      ? recurrence.error
                      : recurrence
                      ? `Se crearán ${recurrence.dates.length} sesiones: ${formatSessionDays(recurrence.dates)}`
                      : ''}
                  </p>
                  <p className="text-[11px] text-muted-foreground">
                    Cada sesión se puede editar o eliminar individualmente después.
                  </p>
                </>
              )}
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
              <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
              {error}
            </div>
          )}

          <Button type="submit" className="w-full" disabled={mutation.isPending}>
            {mutation.isPending ? (
              <LoadingSpinner size="sm" className="border-t-background" />
            ) : activity ? (
              'Guardar cambios'
            ) : (
              'Crear actividad'
            )}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
}

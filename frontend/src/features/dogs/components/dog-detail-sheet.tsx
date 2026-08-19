import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  PawPrint, Shield, Activity, AlertCircle,
  Edit3, Save, X, Ban, Play,
} from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { fetchDogByID, updateDog } from '@/infrastructure/repositories/dog-repository.impl';
import {
  fetchAllIncompatibilities,
  addTraitToDog,
  addTriggerToDog,
  removeIncompatibilityFromDog,
} from '@/infrastructure/repositories/incompatibility-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { SexChip } from '@/components/shared/sex-chip';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useToast } from '@/features/ui/hooks/toast-context';
import { cn } from '@/lib/utils';
import type { Dog } from '@/domain/entities/dog';
import type { ApiError } from '@/infrastructure/api/http-client';

interface DogDetailSheetProps {
  dogId: number;
  onClose: () => void;
}

const SEX_OPTIONS = [
  { value: 'MALE', label: 'Macho' },
  { value: 'FEMALE', label: 'Hembra' },
];

const LEVEL_COLORS: Record<string, string> = {
  ABSOLUTA: 'border-red-500 bg-red-50 dark:bg-red-950/20',
  MEDIA: 'border-amber-500 bg-amber-50 dark:bg-amber-950/20',
  BAJA: 'border-sky-500 bg-sky-50 dark:bg-sky-950/20',
};

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as ApiError;
  const b = apiErr.body as { error?: string; field?: string; details?: string } | null;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

export function DogDetailSheet({ dogId, onClose }: DogDetailSheetProps) {
  const { isAdmin } = useAuth();
  const queryClient = useQueryClient();
  const toast = useToast();
  const [isEditing, setIsEditing] = useState(false);
  const [form, setForm] = useState<Record<string, unknown>>({});
  const [error, setError] = useState('');

  const { data: dog, isLoading } = useQuery({
    queryKey: ['dog', dogId],
    queryFn: () => fetchDogByID(dogId),
  });

  const { data: allIncompats = [] } = useQuery({
    queryKey: ['incompatibilities'],
    queryFn: fetchAllIncompatibilities,
    enabled: isAdmin,
  });

  const availableTraits = allIncompats.filter((i) => i.code && !dog?.traits?.some((t) => t.id === i.id));
  const availableTriggers = allIncompats.filter((i) => i.target_trait_code && !dog?.incompatibilities?.some((t) => t.id === i.id));

  const updateMutation = useMutation({
    mutationFn: () => updateDog(dogId, cleanPatch(form)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dog', dogId] });
      queryClient.invalidateQueries({ queryKey: ['admin-dogs'] });
      queryClient.invalidateQueries({ queryKey: ['active-dogs'] });
      setIsEditing(false);
      setError('');
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al actualizar el perro.')),
  });

  // toggleActiveMutation flips the dog's is_active flag through
  // the same PATCH /dogs/:id endpoint used by the edit form. The
  // backend (admin.PATCH /dogs/:id) accepts a partial patch that
  // already includes is_active (see internal/handler/dog_handler.go
  // modifyDogRequest). We bypass the edit form so the toggle is
  // one click: confirm dialog → mutation → toast. Disabled while
  // editing to avoid racing with the BoolField in the form.
  const toggleActiveMutation = useMutation({
    mutationFn: () => updateDog(dogId, { is_active: !dog?.is_active }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dog', dogId] });
      queryClient.invalidateQueries({ queryKey: ['admin-dogs'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dogs-inactive'] });
      queryClient.invalidateQueries({ queryKey: ['active-dogs'] });
      queryClient.invalidateQueries({ queryKey: ['dog-pass-detail'] });

      const newActive = !dog?.is_active;
      toast.success(
        newActive ? 'Perro activado' : 'Perro desactivado',
        `${dog?.name ?? 'El perro'} ahora está ${newActive ? 'activo' : 'inactivo'}.`,
      );
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al cambiar el estado del perro.')),
  });

  const addTraitMutation = useMutation({
    mutationFn: (traitId: number) => addTraitToDog(dogId, traitId),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['dog', dogId] }); setError(''); },
    onError: (err: unknown) => setError(parseError(err, 'Error al añadir rasgo.')),
  });

  const addTriggerMutation = useMutation({
    mutationFn: (triggerId: number) => addTriggerToDog(dogId, triggerId),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['dog', dogId] }); setError(''); },
    onError: (err: unknown) => setError(parseError(err, 'Error al añadir incompatibilidad.')),
  });

  const removeIncompatMutation = useMutation({
    mutationFn: (incompatId: number) => removeIncompatibilityFromDog(dogId, incompatId),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['dog', dogId] }); setError(''); },
    onError: (err: unknown) => setError(parseError(err, 'Error al eliminar.')),
  });

  function handleSave() {
    setError('');
    updateMutation.mutate();
  }

  function handleCancel() {
    setIsEditing(false);
    setError('');
  }

  function handleToggleActive() {
    if (!dog) return;
    const verb = dog.is_active ? 'desactivar' : 'activar';
    const verbCapitalized = verb.charAt(0).toUpperCase() + verb.slice(1);
    if (confirm(`¿${verbCapitalized} a ${dog.name}?`)) {
      toggleActiveMutation.mutate();
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (!dog) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <PawPrint className="h-10 w-10 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">Perro no encontrado</p>
        <Button variant="outline" size="sm" onClick={onClose}>Volver</Button>
      </div>
    );
  }

  const months = dog.age_in_months;
  const ageText = months < 12 ? `${months} meses` : `${Math.floor(months / 12)} años`;

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex items-center justify-between">
        <button onClick={onClose} className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors">
          ← Volver
        </button>

        {isAdmin && (
          isEditing ? (
            <div className="flex items-center gap-2">
              <Button size="sm" variant="outline" onClick={handleCancel} disabled={updateMutation.isPending}>
                <X className="h-3.5 w-3.5" />
                Cancelar
              </Button>
              <Button size="sm" onClick={handleSave} disabled={updateMutation.isPending}>
                {updateMutation.isPending ? <LoadingSpinner size="sm" className="border-t-background" /> : <Save className="h-3.5 w-3.5" />}
                Guardar
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={handleToggleActive}
                disabled={toggleActiveMutation.isPending}
                className={cn(
                  'gap-1',
                  dog.is_active
                    ? 'border-amber-300 text-amber-700 hover:bg-amber-50 dark:border-amber-800 dark:text-amber-400 dark:hover:bg-amber-950/30'
                    : 'border-emerald-300 text-emerald-700 hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/30',
                )}
              >
                {toggleActiveMutation.isPending ? (
                  <LoadingSpinner size="sm" className="border-t-background" />
                ) : dog.is_active ? (
                  <Ban className="h-3.5 w-3.5" />
                ) : (
                  <Play className="h-3.5 w-3.5" />
                )}
                {dog.is_active ? 'Desactivar' : 'Activar'}
              </Button>
              <Button size="sm" variant="outline" onClick={() => { setForm(buildForm(dog)); setIsEditing(true); }}>
                <Edit3 className="h-3.5 w-3.5" />
                Editar
              </Button>
            </div>
          )
        )}
      </div>

      {error && (
        <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
          {error}
        </div>
      )}

      <div className="flex flex-col items-center gap-4 text-center sm:flex-row sm:text-left">
        <div className="flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-muted">
          {dog.photo_url ? (
            <img src={dog.photo_url} alt={dog.name} className="h-full w-full object-cover" />
          ) : (
            <PawPrint className="h-12 w-12 text-muted-foreground" strokeWidth={1.5} />
          )}
        </div>

        <div>
          <div className="flex items-center justify-center gap-2 sm:justify-start">
            <h1 className="text-2xl font-bold">{dog.name}</h1>
            {!dog.is_active && (
              <span className="rounded bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">Inactivo</span>
            )}
          </div>
          <p className="text-muted-foreground">{dog.breed}</p>
          <div className="mt-2 flex flex-wrap justify-center gap-1.5 sm:justify-start">
            <SexChip sex={dog.sex} />
            {dog.neutered && (
              <span className="inline-flex items-center gap-1 rounded-md bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-300">
                Castrado
              </span>
            )}
            {dog.heat && (
              <span className="inline-flex items-center gap-1 rounded-md bg-pink-100 px-2 py-0.5 text-xs font-medium text-pink-700 dark:bg-pink-900/30 dark:text-pink-300">
                En celo
              </span>
            )}
            <span className="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">{ageText}</span>
            <span className="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">{dog.weight_kg} kg</span>
          </div>
        </div>
      </div>

      <Separator />

      <Section title="Información">
        <Field label="Nombre" value={dog.name} editing={isEditing} form={form} setForm={setForm} field="name" />
        <Field label="Raza" value={dog.breed} editing={isEditing} form={form} setForm={setForm} field="breed" />
        <Field label="Pasaporte" value={dog.passport} editing={isEditing} form={form} setForm={setForm} field="passport" />
        <Field label="Edad (meses)" value={String(ageText)} editing={isEditing} form={form} setForm={setForm} field="age_in_months" type="number" />
        <Field label="Peso (kg)" value={`${dog.weight_kg} kg`} editing={isEditing} form={form} setForm={setForm} field="weight_kg" type="number" />
        <Field label="Foto URL" value={dog.photo_url || '—'} editing={isEditing} form={form} setForm={setForm} field="photo_url" placeholder="https://..." />
      </Section>

      {isEditing && (
        <Section title="Estado">
          <SelectField label="Sexo" value={dog.sex} opts={SEX_OPTIONS} editing form={form} setForm={setForm} field="sex" />
          <BoolField label="Castrado" value={dog.neutered} editing form={form} setForm={setForm} field="neutered" />
          <BoolField label="En celo" value={dog.heat} editing form={form} setForm={setForm} field="heat" />
          <BoolField label="Activo" value={dog.is_active} editing form={form} setForm={setForm} field="is_active" />
        </Section>
      )}

      <Section title="Notas">
        <Field label="Notas médicas" value={dog.medical_notes || '—'} editing={isEditing} form={form} setForm={setForm} field="medical_notes" placeholder="Sin notas" />
        <Field label="Notas educador" value={dog.educator_notes || '—'} editing={isEditing} form={form} setForm={setForm} field="educator_notes" placeholder="Sin notas" />
      </Section>

      {/* Traits */}
      <div>
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">Rasgos</h3>
        <div className="flex flex-wrap items-center gap-1.5">
          {dog.traits.map((t) => (
            <span key={t.id} className={`inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium ${LEVEL_COLORS[t.level] || 'border-border'}`}>
              <Shield className="h-3 w-3" />
              {t.name}
              {isAdmin && (
                <button
                  onClick={() => removeIncompatMutation.mutate(t.id)}
                  className="ml-0.5 rounded-full p-0.5 hover:bg-destructive/20 transition-colors"
                  disabled={removeIncompatMutation.isPending}
                  title="Eliminar rasgo"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </span>
          ))}
          {isAdmin && availableTraits.length > 0 && (
            <div className="flex items-center gap-1">
              <LoadingSpinner size="sm" className={cn(!addTraitMutation.isPending && 'hidden')} />
              <select
                className={`rounded-lg border border-input bg-transparent px-2 py-0.5 text-xs ${addTraitMutation.isPending ? 'hidden' : ''}`}
                onChange={(e) => { if (e.target.value) addTraitMutation.mutate(+e.target.value); }}
                defaultValue=""
                disabled={addTraitMutation.isPending}
              >
                <option value="" disabled>+ Añadir rasgo</option>
                {availableTraits.map((t) => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
            </div>
          )}
          {!isAdmin && dog.traits.length === 0 && (
            <p className="text-xs text-muted-foreground">Sin rasgos registrados</p>
          )}
        </div>
      </div>

      {/* Incompatibilities */}
      <div>
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">Incompatibilidades</h3>
        <div className="flex flex-wrap items-center gap-1.5">
          {dog.incompatibilities.map((inc) => (
            <span key={inc.id} className={`inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium ${LEVEL_COLORS[inc.level] || 'border-border'}`}>
              <Activity className="h-3 w-3" />
              {inc.name}
              {isAdmin && (
                <button
                  onClick={() => removeIncompatMutation.mutate(inc.id)}
                  className="ml-0.5 rounded-full p-0.5 hover:bg-destructive/20 transition-colors"
                  disabled={removeIncompatMutation.isPending}
                  title="Eliminar incompatibilidad"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </span>
          ))}
          {isAdmin && availableTriggers.length > 0 && (
            <div className="flex items-center gap-1">
              <LoadingSpinner size="sm" className={cn(!addTriggerMutation.isPending && 'hidden')} />
              <select
                className={`rounded-lg border border-input bg-transparent px-2 py-0.5 text-xs ${addTriggerMutation.isPending ? 'hidden' : ''}`}
                onChange={(e) => { if (e.target.value) addTriggerMutation.mutate(+e.target.value); }}
                defaultValue=""
                disabled={addTriggerMutation.isPending}
              >
                <option value="" disabled>+ Añadir incompatibilidad</option>
                {availableTriggers.map((t) => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
            </div>
          )}
          {!isAdmin && dog.incompatibilities.length === 0 && (
            <p className="text-xs text-muted-foreground">Sin incompatibilidades registradas</p>
          )}
        </div>
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</h3>
      <div className="grid gap-3 sm:grid-cols-2">{children}</div>
    </div>
  );
}

function Field({
  label, value, editing, form, setForm, field, type = 'text', placeholder,
}: {
  label: string; value: string; editing: boolean;
  form: Record<string, unknown>; setForm: (f: Record<string, unknown>) => void;
  field: string; type?: string; placeholder?: string;
}) {
  if (!editing) {
    return (
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm">{value}</p>
      </div>
    );
  }
  return (
    <div className="space-y-1">
      <label className="text-xs font-medium">{label}</label>
      <input
        className="w-full rounded-lg border border-input bg-transparent px-2.5 py-1.5 text-sm"
        type={type}
        value={form[field] !== undefined ? String(form[field]) : ''}
        placeholder={placeholder}
        onChange={(e) => {
          const val = type === 'number' ? (e.target.value === '' ? '' : +e.target.value) : e.target.value;
          setForm({ ...form, [field]: val });
        }}
      />
    </div>
  );
}

function SelectField({
  label, value, opts, editing, form, setForm, field,
}: {
  label: string; value: string; opts: { value: string; label: string }[];
  editing: boolean; form: Record<string, unknown>; setForm: (f: Record<string, unknown>) => void; field: string;
}) {
  if (!editing) {
    return (
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm">{value}</p>
      </div>
    );
  }
  return (
    <div className="space-y-1">
      <label className="text-xs font-medium">{label}</label>
      <select
        className="w-full rounded-lg border border-input bg-transparent px-2.5 py-1.5 text-sm"
        value={String(form[field] ?? value)}
        onChange={(e) => setForm({ ...form, [field]: e.target.value })}
      >
        {opts.map((o) => (<option key={o.value} value={o.value}>{o.label}</option>))}
      </select>
    </div>
  );
}

function BoolField({
  label, value, editing, form, setForm, field,
}: {
  label: string; value: boolean; editing: boolean;
  form: Record<string, unknown>; setForm: (f: Record<string, unknown>) => void; field: string;
}) {
  const current = form[field] !== undefined ? form[field] : value;
  return (
    <div className="flex items-center gap-3">
      <input
        type="checkbox"
        id={`edit-${field}`}
        checked={!!current}
        onChange={(e) => setForm({ ...form, [field]: e.target.checked })}
        className="h-4 w-4 rounded border-input"
        disabled={!editing}
      />
      <label htmlFor={`edit-${field}`} className={cn('text-sm', !editing && 'text-muted-foreground')}>{label}</label>
    </div>
  );
}

function buildForm(dog: Dog): Record<string, unknown> {
  return {
    name: dog.name, breed: dog.breed, age_in_months: dog.age_in_months, sex: dog.sex,
    neutered: dog.neutered, heat: dog.heat, weight_kg: dog.weight_kg, photo_url: dog.photo_url,
    medical_notes: dog.medical_notes, educator_notes: dog.educator_notes, passport: dog.passport,
    is_active: dog.is_active,
  };
}

function cleanPatch(form: Record<string, unknown>): Record<string, unknown> {
  const patch: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(form)) {
    if (v !== '' && v !== undefined) patch[k] = v;
  }
  return patch;
}

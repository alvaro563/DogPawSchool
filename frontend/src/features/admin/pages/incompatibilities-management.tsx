import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Shield, Activity, AlertCircle, Plus } from 'lucide-react';
import { fetchAllIncompatibilities, createIncompatibility } from '@/infrastructure/repositories/incompatibility-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useToast } from '@/features/ui/hooks/toast-context';
import { cn } from '@/lib/utils';
import type { ApiError } from '@/infrastructure/api/http-client';

const LEVEL_OPTIONS = [
  { value: 'ABSOLUTA', label: 'Absoluta' },
  { value: 'MEDIA', label: 'Media' },
  { value: 'BAJA', label: 'Baja' },
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

export function IncompatibilitiesManagementPage() {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [kind, setKind] = useState<'trait' | 'trigger'>('trait');
  const [name, setName] = useState('');
  const [level, setLevel] = useState('ABSOLUTA');
  const [code, setCode] = useState('');
  const [targetCode, setTargetCode] = useState('');
  const [error, setError] = useState('');

  const { data: incompats = [], isLoading } = useQuery({
    queryKey: ['incompatibilities'],
    queryFn: fetchAllIncompatibilities,
  });

  const traits = useMemo(() => incompats.filter((i) => i.code), [incompats]);
  const triggers = useMemo(() => incompats.filter((i) => i.target_trait_code), [incompats]);

  const createMutation = useMutation({
    mutationFn: () =>
      createIncompatibility({
        name,
        level,
        ...(kind === 'trait' ? { code } : { target_trait_code: targetCode }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['incompatibilities'] });
      setName('');
      setCode('');
      setTargetCode('');
      setError('');

      const kindLabel = kind === 'trait' ? 'Rasgo' : 'Incompatibilidad';
      toast.success(`${kindLabel} creado`, `${name} ya está disponible para asignar a perros.`);
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al crear la incompatibilidad.')),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name || (kind === 'trait' && !code) || (kind === 'trigger' && !targetCode)) {
      setError('Completa todos los campos obligatorios');
      return;
    }
    setError('');
    createMutation.mutate();
  }

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Rasgos e incompatibilidades</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Gestiona los rasgos e incompatibilidades asignables a los perros
        </p>
      </div>

      {/* Create form */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
          <Plus className="h-4 w-4" /> Nuevo rasgo / incompatibilidad
        </h3>

        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="flex items-center rounded-lg border border-border p-0.5 w-fit">
            <button
              type="button"
              onClick={() => setKind('trait')}
              className={cn(
                'rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
                kind === 'trait' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground',
              )}
            >
              Rasgo
            </button>
            <button
              type="button"
              onClick={() => setKind('trigger')}
              className={cn(
                'rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
                kind === 'trigger' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground',
              )}
            >
              Trigger
            </button>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Nombre</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Nivel</label>
              <select className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={level} onChange={(e) => setLevel(e.target.value)}>
                {LEVEL_OPTIONS.map((l) => <option key={l.value} value={l.value}>{l.label}</option>)}
              </select>
            </div>
          </div>

          {kind === 'trait' ? (
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Código del rasgo</label>
              <input
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm font-mono"
                placeholder="Ej: MACHO_ENTERO"
                value={code}
                onChange={(e) => setCode(e.target.value.toUpperCase())}
                required
              />
              <p className="text-[10px] text-muted-foreground">Código único en mayúsculas. Se usará para reglas de compatibilidad.</p>
            </div>
          ) : (
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Rasgo objetivo</label>
              <select
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm"
                value={targetCode}
                onChange={(e) => setTargetCode(e.target.value)}
                required
              >
                <option value="">Seleccionar rasgo existente</option>
                {traits.map((t) => (
                  <option key={t.id} value={t.code}>{t.name} ({t.code})</option>
                ))}
              </select>
              <p className="text-[10px] text-muted-foreground">Este trigger se activará cuando un perro tenga el rasgo seleccionado.</p>
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
              <AlertCircle className="mt-0.5 h-3 w-3 flex-shrink-0" />
              {error}
            </div>
          )}

          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? <LoadingSpinner size="sm" className="border-t-background" /> : 'Crear'}
          </Button>
        </form>
      </div>

      <Separator />

      {/* List */}
      {isLoading ? (
        <div className="flex items-center justify-center py-10"><LoadingSpinner size="lg" /></div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Traits */}
          <div>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
              <Shield className="h-4 w-4 text-sky-500" /> Rasgos ({traits.length})
            </h3>
            {traits.length === 0 ? (
              <p className="text-xs text-muted-foreground">No hay rasgos creados</p>
            ) : (
              <div className="space-y-1.5">
                {traits.map((t) => (
                  <div key={t.id} className="flex items-center justify-between rounded-lg border border-border p-3">
                    <div>
                      <p className="text-sm font-semibold">{t.name}</p>
                      <span className={`inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[10px] font-medium ${LEVEL_COLORS[t.level || '']}`}>
                        {t.level}
                      </span>
                      <code className="ml-2 text-[10px] text-muted-foreground font-mono">{t.code}</code>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Triggers */}
          <div>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
              <Activity className="h-4 w-4 text-amber-500" /> Triggers ({triggers.length})
            </h3>
            {triggers.length === 0 ? (
              <p className="text-xs text-muted-foreground">No hay triggers creados</p>
            ) : (
              <div className="space-y-1.5">
                {triggers.map((t) => (
                  <div key={t.id} className="flex items-center justify-between rounded-lg border border-border p-3">
                    <div>
                      <p className="text-sm font-semibold">{t.name}</p>
                      <span className={`inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[10px] font-medium ${LEVEL_COLORS[t.level || '']}`}>
                        {t.level}
                      </span>
                      <code className="ml-2 text-[10px] text-muted-foreground font-mono">{t.target_trait_code}</code>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

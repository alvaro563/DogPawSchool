import { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Ticket, User, Check, X } from 'lucide-react';
import { fetchAllPasses, fetchPassesByPaid, setPassPaid } from '@/infrastructure/repositories/pass-repository.impl';
import { fetchAllUsers } from '@/infrastructure/repositories/user-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { useToast } from '@/features/ui/hooks/toast-context';
import { formatPrice } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { ApiError } from '@/infrastructure/api/http-client';
import type { Pass } from '@/domain/entities/pass';

type FilterMode = 'all' | 'paid' | 'unpaid';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as ApiError;
  const b = apiErr.body as { error?: string; details?: string } | null;
  if (b?.error === 'validation') return b.details || 'valor inválido';
  if (b?.details) return b.details;
  if (b?.error === 'not_found') return 'Bono no encontrado';
  return fallback;
}

export function PassesManagementPage() {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [filter, setFilter] = useState<FilterMode>('all');

  const { data: passes = [], isLoading } = useQuery({
    queryKey: ['passes', filter],
    queryFn: () => {
      if (filter === 'paid') return fetchPassesByPaid(true);
      if (filter === 'unpaid') return fetchPassesByPaid(false);
      return fetchAllPasses();
    },
  });

  const { data: users = [] } = useQuery({
    queryKey: ['all-users'],
    queryFn: fetchAllUsers,
  });

  const ownerMap = useMemo(() => {
    const m = new Map<number, string>();
    for (const u of users) m.set(u.id, u.name);
    return m;
  }, [users]);

  const togglePaidMutation = useMutation({
    mutationFn: (p: Pass) => setPassPaid(p.id, !p.is_paid),
    onSuccess: (updated, original) => {
      queryClient.invalidateQueries({ queryKey: ['passes'] });
      toast.success(
        updated.is_paid ? 'Bono marcado como pagado' : 'Bono marcado como pendiente',
        `${original.num_of_sessions - original.remaining_sessions}/${original.num_of_sessions} sesiones usadas.`,
      );
    },
    onError: (err: unknown) => {
      toast.error('Error al actualizar el bono', parseError(err, 'Inténtalo de nuevo.'));
    },
  });

  // Partition into active vs finished so the header can show the
  // active count and the body can render a "Bonos finalizados"
  // separator. The criteria (remaining_sessions <= 0 OR expired)
  // mirror admin-dashboard.tsx so both surfaces agree.
  const { activePasses, finishedPasses } = useMemo(() => {
    const active: Pass[] = [];
    const finished: Pass[] = [];
    const now = new Date();
    for (const p of passes) {
      const isExpired = !!p.expires_at && new Date(p.expires_at) < now;
      const isExhausted = p.remaining_sessions <= 0;
      if (isExpired || isExhausted) finished.push(p);
      else active.push(p);
    }
    return { activePasses: active, finishedPasses: finished };
  }, [passes]);

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Bonos</h1>
          <p className="mt-1 text-sm text-muted-foreground">{activePasses.length} bonos activos</p>
        </div>

        <div className="flex items-center rounded-lg border border-border p-0.5">
          {([
            { key: 'all' as const, label: 'Todos' },
            { key: 'paid' as const, label: 'Pagados' },
            { key: 'unpaid' as const, label: 'Pendientes' },
          ]).map((f) => (
            <button
              key={f.key}
              onClick={() => setFilter(f.key)}
              className={cn(
                'rounded-md px-3 py-1.5 text-xs font-medium transition-colors sm:text-sm',
                filter === f.key
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {passes.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <Ticket className="h-10 w-10 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            {filter === 'paid'
              ? 'No hay bonos pagados'
              : filter === 'unpaid'
              ? 'No hay bonos pendientes de pago'
              : 'No hay pases registrados'}
          </p>
        </div>
      ) : (
        <div className="space-y-6">
          {activePasses.length > 0 && (
            <section>
              <div className="space-y-2">
                {activePasses.map((p) => renderPass(p, ownerMap, togglePaidMutation))}
              </div>
            </section>
          )}

          {finishedPasses.length > 0 && (
            <section>
              <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                Bonos finalizados
              </h2>
              <div className="space-y-2">
                {finishedPasses.map((p) => renderPass(p, ownerMap, togglePaidMutation))}
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  );
}

// renderPass renders a single pass row. Extracted from the inline
// .map so the active and finished sections share the exact same
// markup. ownerMap is passed in to avoid rebuilding it on every
// render.
function renderPass(
  p: Pass,
  ownerMap: Map<number, string>,
  togglePaidMutation: { mutate: (p: Pass) => void; isPending: boolean },
) {
  const pct = Math.round((p.remaining_sessions / p.num_of_sessions) * 100);
  const isExpired = !!p.expires_at && new Date(p.expires_at) < new Date();
  const isExhausted = p.remaining_sessions <= 0;

  return (
    <div key={p.id} className={`flex flex-col gap-2 rounded-xl border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between ${isExpired || isExhausted ? 'opacity-50' : ''}`}>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground">
            <Ticket className="h-3 w-3" /> {p.pass_type === 'GENERICO' ? 'Genérico' : 'Específico'}
          </span>
          {isExpired && <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] text-red-700 dark:bg-red-900/30 dark:text-red-300">Expirado</span>}
          {isExhausted && !isExpired && <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">Agotado</span>}
          {p.is_paid ? (
            <span className="inline-flex items-center gap-1 rounded-md bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
              <Check className="h-3 w-3" /> Pagado
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
              Pendiente de pago
            </span>
          )}
        </div>
        <div className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
          <User className="h-3 w-3" />{ownerMap.get(p.user_id) || `ID ${p.user_id}`}
        </div>
      </div>
      <div className="flex items-center gap-3 shrink-0">
        <div className="flex items-center gap-1.5">
          <div className="h-2 w-16 overflow-hidden rounded-full bg-muted">
            <div className={`h-full rounded-full ${pct > 50 ? 'bg-sky-500' : pct > 20 ? 'bg-amber-500' : 'bg-red-500'}`} style={{ width: `${pct}%` }} />
          </div>
          <span className="text-xs tabular-nums text-muted-foreground">{p.remaining_sessions}/{p.num_of_sessions}</span>
        </div>
        <span className="text-xs text-muted-foreground">{formatPrice(p.price)}</span>
        <button
          onClick={() => togglePaidMutation.mutate(p)}
          disabled={togglePaidMutation.isPending}
          className={cn(
            'inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors disabled:opacity-50',
            p.is_paid
              ? 'border border-amber-300 text-amber-700 hover:bg-amber-50 dark:border-amber-800 dark:text-amber-400 dark:hover:bg-amber-950/30'
              : 'border border-emerald-300 text-emerald-700 hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/30',
          )}
          title={p.is_paid ? 'Marcar como pendiente' : 'Marcar como pagado'}
        >
          {p.is_paid ? <X className="h-3 w-3" /> : <Check className="h-3 w-3" />}
          {p.is_paid ? 'Desmarcar' : 'Marcar pagado'}
        </button>
      </div>
    </div>
  );
}

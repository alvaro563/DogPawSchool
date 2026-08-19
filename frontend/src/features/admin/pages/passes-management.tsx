import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Ticket, User } from 'lucide-react';
import { fetchAllPasses } from '@/infrastructure/repositories/pass-repository.impl';
import { fetchAllUsers } from '@/infrastructure/repositories/user-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { formatPrice } from '@/lib/format';

export function PassesManagementPage() {
  const { data: passes = [], isLoading } = useQuery({
    queryKey: ['all-passes'],
    queryFn: fetchAllPasses,
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

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Pases</h1>
        <p className="mt-1 text-sm text-muted-foreground">{passes.length} pases registrados</p>
      </div>

      {passes.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <Ticket className="h-10 w-10 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No hay pases registrados</p>
        </div>
      ) : (
        <div className="space-y-2">
          {passes.map((p) => {
            const pct = Math.round((p.remaining_sessions / p.num_of_sessions) * 100);
            const isExpired = p.expires_at && new Date(p.expires_at) < new Date();
            const isExhausted = p.remaining_sessions <= 0;

            return (
              <div key={p.id} className={`flex flex-col gap-2 rounded-xl border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between ${isExpired || isExhausted ? 'opacity-50' : ''}`}>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground">
                      <Ticket className="h-3 w-3" /> {p.pass_type === 'GENERICO' ? 'Genérico' : 'Específico'}
                    </span>
                    {isExpired && <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] text-red-700 dark:bg-red-900/30 dark:text-red-300">Expirado</span>}
                    {isExhausted && !isExpired && <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">Agotado</span>}
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
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

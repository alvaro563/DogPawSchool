import { useQuery } from '@tanstack/react-query';
import { Ticket, Calendar } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { fetchPassesByUser } from '@/infrastructure/repositories/pass-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';

export function PassListPage() {
  const { user } = useAuth();

  const { data: passes = [], isLoading, error } = useQuery({
    queryKey: ['passes', user?.id],
    queryFn: () => fetchPassesByUser(user!.id),
    enabled: !!user,
  });

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-20">
        <p className="text-sm text-destructive">Error al cargar tus bonos</p>
        <p className="text-xs text-muted-foreground">Inténtalo de nuevo más tarde</p>
      </div>
    );
  }

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Mis Bonos</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {passes.length} {passes.length === 1 ? 'bono disponible' : 'bonos disponibles'}
        </p>
      </div>

      {passes.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <Ticket className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">Aún no tienes bonos</p>
          <p className="text-sm text-muted-foreground">Contacta con la escuela para adquirir bonos</p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {passes.map((pass) => {
            const pct = Math.round((pass.remaining_sessions / pass.num_of_sessions) * 100);
            const isExpired = pass.expires_at && new Date(pass.expires_at) < new Date();
            const isExhausted = pass.remaining_sessions <= 0;

            return (
              <div key={pass.id} className={`rounded-xl border border-border bg-card p-5 ${isExpired || isExhausted ? 'opacity-50' : ''}`}>
                <div className="flex items-center gap-2 mb-3">
                  <span className="inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground">
                    <Ticket className="h-3 w-3" />
                    {pass.pass_type === 'GENERICO' ? 'Genérico' : 'Específico'}
                  </span>
                  {isExpired && <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] font-medium text-red-700 dark:bg-red-900/30 dark:text-red-300">Expirado</span>}
                  {isExhausted && !isExpired && <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">Agotado</span>}
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">Sesiones</span>
                    <span className="font-semibold tabular-nums">{pass.remaining_sessions} / {pass.num_of_sessions}</span>
                  </div>
                  <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className={`h-full rounded-full transition-all ${pct > 50 ? 'bg-sky-500' : pct > 20 ? 'bg-amber-500' : 'bg-red-500'}`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>Precio: {pass.price} cént.</span>
                    <span className="flex items-center gap-1">
                      <Calendar className="h-3 w-3" />
                      {pass.expires_at ? new Date(pass.expires_at).toLocaleDateString('es-ES') : 'Sin caducidad'}
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

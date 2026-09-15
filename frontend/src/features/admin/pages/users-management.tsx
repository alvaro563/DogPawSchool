import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { UserCheck, UserX } from 'lucide-react';
import { fetchAllUsers, deactivateUser, activateUser } from '@/infrastructure/repositories/user-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { useToast } from '@/features/ui/hooks/toast-context';
import { cn } from '@/lib/utils';

export function UsersManagementPage() {
  const queryClient = useQueryClient();
  const toast = useToast();

  const { data: users = [], isLoading } = useQuery({
    queryKey: ['all-users'],
    queryFn: fetchAllUsers,
  });

  const deactivateMutation = useMutation({
    mutationFn: (id: number) => deactivateUser(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['all-users'] });
      const target = users.find((u) => u.id === id);
      const name = target?.name ?? `Usuario #${id}`;
      toast.success('Usuario desactivado', `${name} ya no puede iniciar sesión.`);
    },
  });

  const activateMutation = useMutation({
    mutationFn: (id: number) => activateUser(id),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['all-users'] });
      const target = users.find((u) => u.id === id);
      const name = target?.name ?? `Usuario #${id}`;
      toast.success('Usuario activado', `${name} ahora puede iniciar sesión.`);
    },
  });

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Usuarios</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {users.length} usuarios registrados
        </p>
      </div>

      <div className="overflow-hidden rounded-xl border border-border">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/50">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Nombre</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground hidden sm:table-cell">Email</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Rol</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Estado</th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">Acción</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {users.map((u) => (
                <tr key={u.id} className={cn(!u.is_active && 'opacity-50')}>
                  <td className="px-4 py-3 font-medium">{u.name}</td>
                  <td className="px-4 py-3 text-muted-foreground hidden sm:table-cell">{u.email}</td>
                  <td className="px-4 py-3">
                    <span className={cn(
                      'inline-flex rounded-md px-2 py-0.5 text-xs font-medium',
                      u.role === 'ADMIN' ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300' : 'bg-muted text-muted-foreground',
                    )}>
                      {u.role}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {u.is_active ? (
                      <span className="inline-flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400"><UserCheck className="h-3 w-3" /> Activo</span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-xs text-muted-foreground"><UserX className="h-3 w-3" /> Inactivo</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {u.is_active && u.role !== 'ADMIN' && (
                      <button
                        onClick={() => { if (confirm(`¿Desactivar a ${u.name}?`)) deactivateMutation.mutate(u.id); }}
                        disabled={deactivateMutation.isPending}
                        className="rounded-md px-2 py-1 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 dark:hover:bg-red-950/20 disabled:opacity-50"
                      >
                        Desactivar
                      </button>
                    )}
                    {!u.is_active && u.role !== 'ADMIN' && (
                      <button
                        onClick={() => { if (confirm(`¿Activar a ${u.name}?`)) activateMutation.mutate(u.id); }}
                        disabled={activateMutation.isPending}
                        className="rounded-md px-2 py-1 text-xs font-medium text-emerald-600 transition-colors hover:bg-emerald-50 dark:hover:bg-emerald-950/20 disabled:opacity-50"
                      >
                        Activar
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

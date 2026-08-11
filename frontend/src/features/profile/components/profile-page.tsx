import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { User, Edit3, Save, X, AlertCircle } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { fetchUserByID, updateUser } from '@/infrastructure/repositories/user-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import type { ApiError } from '@/infrastructure/api/http-client';

function parseError(err: unknown, fallback: string): string {
  const apiErr = err as ApiError;
  const b = apiErr.body as { error?: string; field?: string; details?: string } | null;
  if (b?.error === 'validation' && b?.field) {
    return `Error en ${b.field}: ${b.details || 'valor inválido'}`;
  }
  if (b?.details) return b.details;
  return fallback;
}

export function ProfilePage() {
  const { user, isAdmin } = useAuth();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');

  const { data: profile, isLoading } = useQuery({
    queryKey: ['user', user?.id],
    queryFn: () => fetchUserByID(user!.id),
    enabled: !!user,
  });

  const updateMutation = useMutation({
    mutationFn: () => updateUser(user!.id, { name, email }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user', user?.id] });
      setEditing(false);
      setError('');
    },
    onError: (err: unknown) => setError(parseError(err, 'Error al actualizar el perfil.')),
  });

  function handleEdit() {
    if (profile) {
      setName(profile.name);
      setEmail(profile.email);
    }
    setEditing(true);
  }

  function handleSave() {
    setError('');
    updateMutation.mutate();
  }

  function handleCancel() {
    setEditing(false);
    setError('');
  }

  if (isLoading) return <div className="flex items-center justify-center py-20"><LoadingSpinner size="lg" /></div>;
  if (!profile) return <p className="py-20 text-center text-sm text-muted-foreground">Perfil no encontrado</p>;

  return (
    <div className="mx-auto max-w-lg px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Perfil</h1>
        {isAdmin && !editing && (
          <Button size="sm" variant="outline" onClick={handleEdit}>
            <Edit3 className="h-3.5 w-3.5" /> Editar
          </Button>
        )}
        {editing && (
          <div className="flex items-center gap-2">
            <Button size="sm" variant="outline" onClick={handleCancel} disabled={updateMutation.isPending}>
              <X className="h-3.5 w-3.5" /> Cancelar
            </Button>
            <Button size="sm" onClick={handleSave} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? <LoadingSpinner size="sm" className="border-t-background" /> : <><Save className="h-3.5 w-3.5" /> Guardar</>}
            </Button>
          </div>
        )}
      </div>

      {error && (
        <div className="mb-4 flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />{error}
        </div>
      )}

      <div className="flex flex-col items-center gap-4 mb-6">
        <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-muted">
          <User className="h-10 w-10 text-muted-foreground" strokeWidth={1.5} />
        </div>
        <div className="text-center">
          <p className="text-xl font-bold">{profile.name}</p>
          <p className="text-sm text-muted-foreground">{profile.email}</p>
        </div>
      </div>

      <Separator />

      <div className="mt-6 space-y-4">
        {editing ? (
          <>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Nombre</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">Email</label>
              <input className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
          </>
        ) : (
          <div className="grid gap-3">
            <div>
              <p className="text-xs text-muted-foreground">Nombre</p>
              <p className="text-sm">{profile.name}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Email</p>
              <p className="text-sm">{profile.email}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Rol</p>
              <span className={`inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium ${profile.role === 'ADMIN' ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300' : 'bg-muted text-muted-foreground'}`}>
                {profile.role}
              </span>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Estado</p>
              <span className={`inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium ${profile.is_active ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'}`}>
                {profile.is_active ? 'Activo' : 'Inactivo'}
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

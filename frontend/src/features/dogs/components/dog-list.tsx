import { useQuery } from '@tanstack/react-query';
import { PawPrint, VenetianMask } from 'lucide-react';
import { useNavigate } from '@tanstack/react-router';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { fetchDogsByOwner } from '@/infrastructure/repositories/dog-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { SexChip } from '@/components/shared/sex-chip';
import type { Dog } from '@/domain/entities/dog';

function DogCard({ dog }: { dog: Dog }) {
  const navigate = useNavigate();
  const months = dog.age_in_months;
  const ageText = months < 12 ? `${months} meses` : `${Math.floor(months / 12)} años`;

  return (
    <button
      onClick={() => navigate({ to: '/dog-detail/$id', params: { id: String(dog.id) } })}
      className={`w-full rounded-xl border border-border bg-card p-5 text-left transition-shadow hover:shadow-md ${
        !dog.is_active ? 'opacity-50' : ''
      }`}
    >
      <div className="flex items-center gap-4">
        <div className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted">
          {dog.photo_url ? (
            <img
              src={dog.photo_url}
              alt={dog.name}
              className="h-full w-full object-cover"
            />
          ) : (
            <PawPrint className="h-7 w-7 text-muted-foreground" strokeWidth={1.5} />
          )}
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h3 className="truncate text-lg font-bold">{dog.name}</h3>
            {!dog.is_active && (
              <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                Inactivo
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">{dog.breed}</p>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap gap-1.5">
        <SexChip sex={dog.sex} />

        {dog.neutered && (
          <span className="inline-flex items-center gap-1 rounded-md bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-300">
            Esterilizado
          </span>
        )}

        {dog.heat && (
          <span className="inline-flex items-center gap-1 rounded-md bg-pink-100 px-2 py-0.5 text-xs font-medium text-pink-700 dark:bg-pink-900/30 dark:text-pink-300">
            En celo
          </span>
        )}

        <span className="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
          {ageText}
        </span>

        <span className="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
          {dog.weight_kg} kg
        </span>
      </div>

      <p className="mt-3 flex items-center gap-1.5 truncate text-xs text-muted-foreground">
        <VenetianMask className="h-3 w-3" />
        {dog.passport}
      </p>
    </button>
  );
}

export function DogListPage() {
  const { user } = useAuth();

  const {
    data: dogs = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: ['dogs', user?.id],
    queryFn: () => fetchDogsByOwner(user!.id),
    enabled: !!user,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-20 text-center">
        <p className="text-sm text-destructive">Error al cargar tus perros</p>
        <p className="text-xs text-muted-foreground">Inténtalo de nuevo más tarde</p>
      </div>
    );
  }

  if (dogs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
          <PawPrint className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
        </div>
        <p className="text-lg font-semibold">Aún no tienes perros registrados</p>
        <p className="text-sm text-muted-foreground">
          Contacta con la escuela para dar de alta a tu perro
        </p>
      </div>
    );
  }

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Mis Perros</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {dogs.length} {dogs.length === 1 ? 'perro registrado' : 'perros registrados'}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {dogs.map((dog) => (
          <DogCard key={dog.id} dog={dog} />
        ))}
      </div>
    </div>
  );
}

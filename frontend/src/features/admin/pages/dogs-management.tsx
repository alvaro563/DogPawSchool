import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { PawPrint, Dog, VenetianMask, Search } from 'lucide-react';
import { fetchAllActiveDogs, fetchDogsByNeutered, fetchDogsByHeat } from '@/infrastructure/repositories/dog-repository.impl';
import { fetchAllUsers } from '@/infrastructure/repositories/user-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { cn } from '@/lib/utils';
import { SexChip } from '@/components/shared/sex-chip';

type FilterMode = 'all' | 'neutered' | 'heat';

interface SearchBarProps {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
  suggestions: { key: string; label: string }[];
  onSelect: (v: string) => void;
}

function SearchBar({ label, value, onChange, placeholder, suggestions, onSelect }: SearchBarProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="space-y-1.5">
      <label className="text-xs font-medium">{label}</label>
      <div className="relative">
        <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
        <input
          className="w-full rounded-lg border border-input bg-transparent py-2 pl-8 pr-3 text-sm"
          placeholder={placeholder}
          value={value}
          onChange={(e) => { onChange(e.target.value); setOpen(true); }}
          onFocus={() => setOpen(true)}
          onBlur={() => setTimeout(() => setOpen(false), 150)}
        />
        {open && suggestions.length > 0 && (
          <div className="absolute z-50 mt-1 max-h-48 w-full overflow-y-auto rounded-lg border border-border bg-popover p-1 shadow-lg">
            {suggestions.map((s) => (
              <button
                key={s.key}
                className="w-full rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted truncate"
                onMouseDown={() => { onSelect(s.label); setOpen(false); }}
              >
                {s.label}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export function DogsManagementPage() {
  const navigate = useNavigate();
  const [filter, setFilter] = useState<FilterMode>('all');
  const [searchDog, setSearchDog] = useState('');
  const [searchBreed, setSearchBreed] = useState('');
  const [searchOwner, setSearchOwner] = useState('');

  const { data: dogs = [], isLoading } = useQuery({
    queryKey: ['admin-dogs', filter],
    queryFn: () => {
      if (filter === 'neutered') return fetchDogsByNeutered();
      if (filter === 'heat') return fetchDogsByHeat();
      return fetchAllActiveDogs();
    },
  });

  const { data: users = [] } = useQuery({
    queryKey: ['all-users'],
    queryFn: fetchAllUsers,
  });

  const ownerMap = useMemo(() => {
    const map = new Map<number, string>();
    for (const u of users) map.set(u.id, u.name);
    return map;
  }, [users]);

  const filtered = useMemo(() => {
    const tDog = searchDog.trim().toLowerCase();
    const tBreed = searchBreed.trim().toLowerCase();
    const tOwner = searchOwner.trim().toLowerCase();
    if (!tDog && !tBreed && !tOwner) return dogs;
    return dogs.filter((d) => {
      if (tDog && !d.name.toLowerCase().includes(tDog)) return false;
      if (tBreed && !d.breed.toLowerCase().includes(tBreed)) return false;
      if (tOwner && !(ownerMap.get(d.user_id) || '').toLowerCase().includes(tOwner)) return false;
      return true;
    });
  }, [dogs, searchDog, searchBreed, searchOwner, ownerMap]);

  const dogSuggestions = useMemo(
    () => dogs.slice(0, 20).map((d) => ({ key: `dog-${d.id}`, label: d.name })),
    [dogs],
  );
  const breedSuggestions = useMemo(
    () => [...new Set(dogs.map((d) => d.breed))].sort().map((b) => ({ key: `breed-${b}`, label: b })),
    [dogs],
  );
  const ownerSuggestions = useMemo(
    () =>
      [...new Map(users.filter((u) => dogs.some((d) => d.user_id === u.id)).map((u) => [u.id, u])).values()].map(
        (u) => ({ key: `owner-${u.id}`, label: u.name }),
      ),
    [dogs, users],
  );

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Perros activos
            {!isLoading && (
              <span className="ml-2 text-lg font-normal text-muted-foreground">({dogs.length})</span>
            )}
          </h1>
        </div>

        <div className="flex items-center rounded-lg border border-border p-0.5">
          {([
            { key: 'all' as const, label: 'Todos' },
            { key: 'neutered' as const, label: 'Castrados' },
            { key: 'heat' as const, label: 'En celo' },
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

      <div className="grid gap-3 sm:grid-cols-3">
        <SearchBar
          label="Perro"
          value={searchDog}
          onChange={setSearchDog}
          placeholder="Nombre del perro..."
          suggestions={dogSuggestions}
          onSelect={setSearchDog}
        />
        <SearchBar
          label="Raza"
          value={searchBreed}
          onChange={setSearchBreed}
          placeholder="Raza del perro..."
          suggestions={breedSuggestions}
          onSelect={setSearchBreed}
        />
        <SearchBar
          label="Dueño"
          value={searchOwner}
          onChange={setSearchOwner}
          placeholder="Nombre del dueño..."
          suggestions={ownerSuggestions}
          onSelect={setSearchOwner}
        />
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <LoadingSpinner size="lg" />
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
            <PawPrint className="h-8 w-8 text-muted-foreground" strokeWidth={1.5} />
          </div>
          <p className="text-lg font-semibold">Sin resultados</p>
          <p className="text-sm text-muted-foreground">
            {searchDog || searchBreed || searchOwner
              ? 'Prueba con otros filtros de búsqueda'
              : 'No se encontraron perros con este filtro'}
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((dog) => {
            const months = dog.age_in_months;
            const ageText = months < 12 ? `${months} meses` : `${Math.floor(months / 12)} años`;
            const ownerName = ownerMap.get(dog.user_id) || `ID: ${dog.user_id}`;

            return (
              <button
                key={dog.id}
                onClick={() => navigate({ to: '/dog-detail/$id', params: { id: String(dog.id) } })}
                className={cn(
                  'w-full rounded-xl border border-border bg-card p-5 text-left transition-shadow hover:shadow-md',
                  !dog.is_active && 'opacity-50',
                )}
              >
                <div className="flex items-center gap-4">
                  <div className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted">
                    {dog.photo_url ? (
                      <img src={dog.photo_url} alt={dog.name} className="h-full w-full object-cover" />
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
                      Castrado
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

                <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
                  <span className="flex items-center gap-1.5 truncate">
                    <VenetianMask className="h-3 w-3 shrink-0" />
                    {dog.passport}
                  </span>
                  <span className="flex items-center gap-1 shrink-0">
                    <Dog className="h-3 w-3" />
                    {ownerName}
                  </span>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

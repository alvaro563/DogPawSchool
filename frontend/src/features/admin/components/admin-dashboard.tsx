import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchActivities } from '@/infrastructure/repositories/activity-repository.impl';
import { fetchUserReservations } from '@/infrastructure/repositories/reservation-repository.impl';
import { fetchAllUsers } from '@/infrastructure/repositories/user-repository.impl';
import { fetchActiveDogsCount } from '@/infrastructure/repositories/dog-repository.impl';
import { fetchAllPasses } from '@/infrastructure/repositories/pass-repository.impl';
import { useAdminModal } from '@/features/admin/hooks/admin-modal-context';
import { KpiCards } from './kpi-cards';
import { PendingReservations } from './pending-reservations';
import { QuickActions } from './quick-actions';
import { TodayOperations } from './today-operations';
import type { ReservationView } from '@/domain/entities/reservation';

const TODAY_START = new Date();
TODAY_START.setHours(0, 0, 0, 0);
const TODAY_END = new Date(TODAY_START);
TODAY_END.setDate(TODAY_END.getDate() + 1);

function toISO(d: Date): string {
  return d.toISOString().replace(/\.\d{3}/, '');
}

export function AdminDashboard() {
  const { open } = useAdminModal();

  const { data: todayActivities = [], isLoading: todayLoading } = useQuery({
    queryKey: ['admin-dashboard', 'today-activities'],
    queryFn: () => fetchActivities(toISO(TODAY_START), toISO(TODAY_END)),
  });

  const { data: users = [] } = useQuery({
    queryKey: ['admin-dashboard', 'users'],
    queryFn: fetchAllUsers,
  });

  const { data: activeDogsCount = 0, isLoading: dogsLoading } = useQuery({
    queryKey: ['admin-dashboard', 'active-dogs'],
    queryFn: fetchActiveDogsCount,
  });

  const { data: allPasses = [], isLoading: passesLoading } = useQuery({
    queryKey: ['admin-dashboard', 'all-passes'],
    queryFn: fetchAllPasses,
  });

  const activePasses = useMemo(
    () =>
      allPasses.filter((p) => {
        const notExhausted = p.remaining_sessions > 0;
        const notExpired = !p.expires_at || new Date(p.expires_at) > new Date();
        return notExhausted && notExpired;
      }),
    [allPasses],
  );

  const { data: pendingReservations = [], isLoading: pendingLoading } = useQuery({
    queryKey: ['admin-dashboard', 'pending-reservations', users.map((u) => u.id)],
    queryFn: async () => {
      const results: (ReservationView & { owner_id: number; owner_name: string })[] = [];
      const batchResults = await Promise.allSettled(
        users.map((user) => fetchUserReservations(user.id, 'PENDING_TO_CONFIRM')),
      );
      batchResults.forEach((result, i) => {
        if (result.status === 'fulfilled') {
          for (const r of result.value) {
            results.push({ ...r, owner_id: users[i].id, owner_name: users[i].name });
          }
        }
      });
      return results;
    },
    enabled: users.length > 0,
  });

  const kpiLoading = todayLoading || dogsLoading || passesLoading;

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Panel de administración de Dog Paw
          </p>
        </div>
      </div>

      <KpiCards
        todayClasses={todayActivities.length}
        pendingReservations={pendingReservations.length}
        activeDogs={activeDogsCount}
        activePasses={activePasses.length}
        isLoading={kpiLoading}
      />

      <div className="grid gap-6 lg:grid-cols-2">
        <QuickActions onAction={open} />
        <TodayOperations activities={todayActivities} isLoading={todayLoading} />
      </div>

      <PendingReservations items={pendingReservations} isLoading={pendingLoading} />
    </div>
  );
}

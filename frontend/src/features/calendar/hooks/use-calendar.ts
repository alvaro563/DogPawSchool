import { useQuery } from '@tanstack/react-query';
import { useState, useMemo, useCallback } from 'react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { fetchActivities } from '@/infrastructure/repositories/activity-repository.impl';
import { fetchUserReservations } from '@/infrastructure/repositories/reservation-repository.impl';

export type ViewMode = 'day' | 'week' | 'month';

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function startOfWeek(d: Date): Date {
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1);
  return new Date(d.getFullYear(), d.getMonth(), diff);
}

function startOfMonth(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), 1);
}

function endOfRange(d: Date, mode: ViewMode): Date {
  const s = new Date(d);
  if (mode === 'day') {
    s.setDate(s.getDate() + 1);
  } else if (mode === 'week') {
    s.setDate(s.getDate() + 7);
  } else {
    s.setMonth(s.getMonth() + 1);
  }
  return s;
}

function stepDate(d: Date, mode: ViewMode, direction: 1 | -1): Date {
  const s = new Date(d);
  if (mode === 'day') {
    s.setDate(s.getDate() + direction);
  } else if (mode === 'week') {
    s.setDate(s.getDate() + direction * 7);
  } else {
    s.setMonth(s.getMonth() + direction);
  }
  return s;
}

function toISO(d: Date): string {
  return d.toISOString().replace(/\.\d{3}/, '');
}

export function useCalendar() {
  const { user } = useAuth();
  const [currentDate, setCurrentDate] = useState(() => startOfMonth(new Date()));
  const [viewMode, setViewMode] = useState<ViewMode>('month');

  const rangeStart = useMemo(() => {
    if (viewMode === 'month') return startOfMonth(currentDate);
    if (viewMode === 'week') return startOfWeek(currentDate);
    return startOfDay(currentDate);
  }, [currentDate, viewMode]);

  const rangeEnd = useMemo(() => endOfRange(rangeStart, viewMode), [rangeStart, viewMode]);

  const {
    data: activities = [],
    isLoading: activitiesLoading,
    error: activitiesError,
  } = useQuery({
    queryKey: ['activities', toISO(rangeStart), toISO(rangeEnd)],
    queryFn: () => fetchActivities(toISO(rangeStart), toISO(rangeEnd)),
    enabled: !!user,
  });

  const { data: reservations = [] } = useQuery({
    queryKey: ['reservations', user?.id],
    queryFn: () => fetchUserReservations(user!.id, 'CONFIRMED,PENDING_TO_CONFIRM'),
    enabled: !!user,
  });

  const userReservationMap = useMemo(() => {
    const map = new Map<number, string>();
    for (const r of reservations) {
      const prev = map.get(r.activity_id);
      if (prev === 'CONFIRMED') continue;
      map.set(r.activity_id, r.status);
    }
    return map;
  }, [reservations]);

  const navigate = useCallback(
    (direction: 1 | -1) => {
      setCurrentDate((d) => stepDate(d, viewMode, direction));
    },
    [viewMode],
  );

  const goToday = useCallback(() => {
    setCurrentDate(new Date());
  }, []);

  const setView = useCallback((mode: ViewMode) => {
    setViewMode(mode);
    setCurrentDate(new Date());
  }, []);

  const isLoading = activitiesLoading;

  return {
    currentDate,
    rangeStart,
    rangeEnd,
    viewMode,
    setView,
    navigate,
    goToday,
    activities,
    userReservationMap,
    isLoading,
    error: activitiesError,
  };
}

export function formatActivityTime(date: string, durationInHours: number): string {
  const start = new Date(date);
  const end = new Date(start.getTime() + durationInHours * 3600000);
  const fmt = (d: Date) =>
    d.toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit' });
  return `${fmt(start)} - ${fmt(end)}`;
}

export function isActivityPast(date: string, durationInHours: number): boolean {
  const end = new Date(new Date(date).getTime() + durationInHours * 3600000);
  return end < new Date();
}

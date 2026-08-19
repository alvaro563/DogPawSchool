import { useMemo, useState } from 'react';
import { useCalendar } from '@/features/calendar/hooks/use-calendar';
import { useSelectedActivity } from '@/features/calendar/hooks/selected-activity-context';
import { CalendarHeader } from './calendar-header';
import { MonthView } from './month-view';
import { WeekView } from './week-view';
import { DayView } from './day-view';
import { ActivityDetailSheet } from './activity-detail-sheet';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import type { Activity } from '@/domain/entities/activity';
import type { ReservationView } from '@/domain/entities/reservation';

const SHOWN_STATUSES = new Set(['CONFIRMED', 'PENDING_TO_CONFIRM']);

export interface ExistingReservation {
  dogId: number;
  status: string;
  passId: number;
}

export function CalendarView() {
  const {
    currentDate,
    viewMode,
    setView,
    navigate,
    goToday,
    activities,
    userReservationMap,
    userReservations,
    isLoading,
  } = useCalendar();

  const { activity: contextActivity, setActivity: setContextActivity } = useSelectedActivity();
  const [sheetOpen, setSheetOpen] = useState(false);

  function handleActivityClick(activity: Activity) {
    setContextActivity(activity);
    setSheetOpen(true);
  }

  function handleSheetOpenChange(open: boolean) {
    setSheetOpen(open);
    if (!open) {
      setContextActivity(null);
    }
  }

  // Existing reservations for the currently selected activity,
  // filtered to the two statuses that occupy a slot. Used by the
  // sheet to (a) exclude already-booked dogs from the selector and
  // (b) decide whether the user can book ANOTHER dog for the same
  // activity.
  const existingReservations = useMemo<ExistingReservation[]>(() => {
    if (!contextActivity) return [];
    return userReservations
      .filter(
        (r: ReservationView) =>
          r.activity_id === contextActivity.id && SHOWN_STATUSES.has(r.status),
      )
      .map((r) => ({ dogId: r.dog_id, status: r.status, passId: r.pass_id }));
  }, [contextActivity, userReservations]);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center py-20">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <CalendarHeader
        currentDate={currentDate}
        viewMode={viewMode}
        onViewChange={setView}
        onPrev={() => navigate(-1)}
        onNext={() => navigate(1)}
        onToday={goToday}
      />

      {viewMode === 'month' && (
        <MonthView
          currentDate={currentDate}
          activities={activities}
          userReservationMap={userReservationMap}
          onActivityClick={handleActivityClick}
        />
      )}
      {viewMode === 'week' && (
        <WeekView
          currentDate={currentDate}
          activities={activities}
          userReservationMap={userReservationMap}
          onActivityClick={handleActivityClick}
        />
      )}
      {viewMode === 'day' && (
        <DayView
          currentDate={currentDate}
          activities={activities}
          userReservationMap={userReservationMap}
          onActivityClick={handleActivityClick}
        />
      )}

      <ActivityDetailSheet
        activity={contextActivity}
        existingReservations={existingReservations}
        open={sheetOpen}
        onOpenChange={handleSheetOpenChange}
      />
    </div>
  );
}

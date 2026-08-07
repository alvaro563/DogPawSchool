import { useState } from 'react';
import { useCalendar } from '@/features/calendar/hooks/use-calendar';
import { CalendarHeader } from './calendar-header';
import { MonthView } from './month-view';
import { WeekView } from './week-view';
import { DayView } from './day-view';
import { ActivityDetailSheet } from './activity-detail-sheet';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import type { Activity } from '@/domain/entities/activity';

export function CalendarView() {
  const {
    currentDate,
    viewMode,
    setView,
    navigate,
    goToday,
    activities,
    userReservationMap,
    isLoading,
  } = useCalendar();

  const [selectedActivity, setSelectedActivity] = useState<Activity | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  function handleActivityClick(activity: Activity) {
    setSelectedActivity(activity);
    setSheetOpen(true);
  }

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
        activity={selectedActivity}
        reservationStatus={
          selectedActivity
            ? userReservationMap.get(selectedActivity.id)
            : undefined
        }
        open={sheetOpen}
        onOpenChange={setSheetOpen}
      />
    </div>
  );
}

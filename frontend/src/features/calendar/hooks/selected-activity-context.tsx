import { createContext, useContext, useState, type ReactNode } from 'react';
import type { Activity } from '@/domain/entities/activity';

// SelectedActivityState holds the activity the user has just
// clicked in the calendar. It is shared between the calendar
// (producer — CalendarView sets it) and the sidebar (consumer —
// the "Reservar otro perro en esta actividad" quick action reads
// it to know which activity to target).
//
// Single source of truth for "what activity is the user currently
// looking at". Without this context the sidebar would not know
// which activity to offer the additional-dog form for, because
// that state was previously local to CalendarView.
interface SelectedActivityState {
  activity: Activity | null;
  setActivity: (a: Activity | null) => void;
}

const SelectedActivityContext = createContext<SelectedActivityState | null>(null);

export function SelectedActivityProvider({ children }: { children: ReactNode }) {
  const [activity, setActivity] = useState<Activity | null>(null);

  return (
    <SelectedActivityContext.Provider value={{ activity, setActivity }}>
      {children}
    </SelectedActivityContext.Provider>
  );
}

export function useSelectedActivity(): SelectedActivityState {
  const ctx = useContext(SelectedActivityContext);
  if (!ctx) {
    throw new Error('useSelectedActivity must be used inside SelectedActivityProvider');
  }
  return ctx;
}

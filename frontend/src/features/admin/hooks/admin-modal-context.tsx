import { createContext, useContext, useState, type ReactNode } from 'react';
import type { Activity } from '@/domain/entities/activity';

export type AdminModal = 'activity' | 'pass' | 'dog' | 'client' | null;

interface AdminModalState {
  active: AdminModal;
  // Activity being edited when the 'activity' modal opens in edit
  // mode; null = create mode (the original behaviour).
  editingActivity: Activity | null;
  // Minimum allowed max_capacity while editing: the number of slots
  // already held (confirmed + pending) so a PATCH can never push
  // available_spots negative. 1 (the backend floor) in create mode.
  editingMinCapacity: number;
  open: (modal: AdminModal, activity?: Activity, minCapacity?: number) => void;
  close: () => void;
}

const AdminModalContext = createContext<AdminModalState | null>(null);

export function AdminModalProvider({ children }: { children: ReactNode }) {
  const [active, setActive] = useState<AdminModal>(null);
  const [editingActivity, setEditingActivity] = useState<Activity | null>(null);
  const [editingMinCapacity, setEditingMinCapacity] = useState(1);

  const open = (modal: AdminModal, activity?: Activity, minCapacity?: number) => {
    setEditingActivity(activity ?? null);
    setEditingMinCapacity(minCapacity ?? 1);
    setActive(modal);
  };

  const close = () => {
    setActive(null);
    setEditingActivity(null);
    setEditingMinCapacity(1);
  };

  return (
    <AdminModalContext.Provider value={{ active, editingActivity, editingMinCapacity, open, close }}>
      {children}
    </AdminModalContext.Provider>
  );
}

export function useAdminModal(): AdminModalState {
  const ctx = useContext(AdminModalContext);
  if (!ctx) throw new Error('useAdminModal must be used inside AdminModalProvider');
  return ctx;
}

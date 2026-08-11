import { createContext, useContext, useState, type ReactNode } from 'react';

export type AdminModal = 'activity' | 'pass' | 'dog' | 'client' | null;

interface AdminModalState {
  active: AdminModal;
  open: (modal: AdminModal) => void;
  close: () => void;
}

const AdminModalContext = createContext<AdminModalState | null>(null);

export function AdminModalProvider({ children }: { children: ReactNode }) {
  const [active, setActive] = useState<AdminModal>(null);

  return (
    <AdminModalContext.Provider value={{ active, open: setActive, close: () => setActive(null) }}>
      {children}
    </AdminModalContext.Provider>
  );
}

export function useAdminModal(): AdminModalState {
  const ctx = useContext(AdminModalContext);
  if (!ctx) throw new Error('useAdminModal must be used inside AdminModalProvider');
  return ctx;
}

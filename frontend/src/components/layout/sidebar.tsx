import { Link, useLocation } from '@tanstack/react-router';
import { CalendarDays, Dog, Ticket, ClipboardList, User } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { cn } from '@/lib/utils';

const userNavItems = [
  { to: '/calendar', label: 'Calendario', icon: CalendarDays },
  { to: '/dogs', label: 'Mis Perros', icon: Dog },
  { to: '/passes', label: 'Mis Bonos', icon: Ticket },
  { to: '/reservations', label: 'Mis Reservas', icon: ClipboardList },
  { to: '/profile', label: 'Perfil', icon: User },
] as const;

interface SidebarProps {
  className?: string;
  onNavigate?: () => void;
}

export function Sidebar({ className, onNavigate }: SidebarProps) {
  const { user, isAdmin } = useAuth();
  const location = useLocation();

  const navItems = isAdmin
    ? [{ to: '/admin', label: 'Dashboard', icon: User }]
    : userNavItems;

  return (
    <nav className={cn('flex flex-col gap-1 px-3 py-4', className)}>
      {user && !isAdmin && (
        <div className="mb-4 rounded-lg bg-muted/50 px-3 py-3">
          <p className="text-xs font-medium text-muted-foreground">Hola,</p>
          <p className="truncate text-sm font-semibold">{user.name}</p>
        </div>
      )}

      {navItems.map((item) => {
        const Icon = item.icon;
        const isActive = location.pathname.startsWith(item.to);
        return (
          <Link
            key={item.to}
            to={item.to}
            onClick={onNavigate}
            className={cn(
              'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
              isActive
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
          >
            <Icon className="h-4 w-4" strokeWidth={2} />
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}

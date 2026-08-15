import { Link, useLocation } from '@tanstack/react-router';
import { CalendarDays, Dog, Ticket, ClipboardList, User, LayoutDashboard, Users, School, CreditCard, PlusCircle, Shield } from 'lucide-react';
import { useAuth } from '@/features/auth/hooks/use-auth';
import { useAdminModal } from '@/features/admin/hooks/admin-modal-context';
import { cn } from '@/lib/utils';

interface NavItem {
  to: string;
  label: string;
  icon: React.ComponentType<{ className?: string; strokeWidth?: number }>;
}

const userNavItems: NavItem[] = [
  { to: '/calendar', label: 'Calendario', icon: CalendarDays },
  { to: '/dogs', label: 'Mis Perros', icon: Dog },
  { to: '/passes', label: 'Mis Bonos', icon: Ticket },
  { to: '/reservations', label: 'Mis Reservas', icon: ClipboardList },
  { to: '/profile', label: 'Perfil', icon: User },
];

const adminNavItems: NavItem[] = [
  { to: '/admin', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/calendar', label: 'Calendario', icon: CalendarDays },
  { to: '/admin/users', label: 'Usuarios', icon: Users },
  { to: '/admin/dogs', label: 'Perros', icon: Dog },
  { to: '/admin/activities', label: 'Actividades', icon: School },
  { to: '/admin/passes', label: 'Pases', icon: CreditCard },
  { to: '/admin/reservations', label: 'Reservas', icon: ClipboardList },
];

interface SidebarProps {
  className?: string;
  onNavigate?: () => void;
}

export function Sidebar({ className, onNavigate }: SidebarProps) {
  const { user, isAdmin } = useAuth();
  const location = useLocation();
  const { open } = useAdminModal();

  const navItems = isAdmin ? adminNavItems : userNavItems;

  function handleQuickAction(action: 'activity' | 'pass' | 'dog' | 'client') {
    open(action);
    onNavigate?.();
  }

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
        const isActive = item.to === '/admin'
          ? location.pathname === '/admin' || location.pathname === '/admin/'
          : location.pathname.startsWith(item.to);
        return (
          <Link
            key={item.to}
            to={item.to as '/'}
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

      {isAdmin && (
        <>
          <Separator text="Gestión de escuela" />

          <button
            onClick={() => handleQuickAction('activity')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <PlusCircle className="h-4 w-4" strokeWidth={2} />
            Crear Actividad
          </button>

          <button
            onClick={() => handleQuickAction('pass')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <Ticket className="h-4 w-4" strokeWidth={2} />
            Asignar Bono
          </button>

          <button
            onClick={() => handleQuickAction('dog')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <Dog className="h-4 w-4" strokeWidth={2} />
            Registrar Perro
          </button>

          <button
            onClick={() => handleQuickAction('client')}
            className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <User className="h-4 w-4" strokeWidth={2} />
            Alta Cliente
          </button>

          <Link
            to={'/incompatibilities' as '/'}
            onClick={onNavigate}
            className={cn(
              'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
              location.pathname.startsWith('/incompatibilities')
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
          >
            <Shield className="h-4 w-4" strokeWidth={2} />
            Crear Rasgo / Incompatibilidad
          </Link>
        </>
      )}
    </nav>
  );
}

function Separator({ text }: { text: string }) {
  return (
    <div className="flex items-center gap-2 px-3 py-1">
      <div className="h-px flex-1 bg-border" />
      {text && <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{text}</span>}
      <div className="h-px flex-1 bg-border" />
    </div>
  );
}

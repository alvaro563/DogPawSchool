import { PlusCircle, Ticket, Dog, UserPlus } from 'lucide-react';

interface QuickActionsProps {
  onAction: (action: 'activity' | 'pass' | 'dog' | 'client') => void;
}

const actions = [
  { key: 'activity' as const, label: 'Crear Actividad', desc: 'Programar nueva clase o ruta', icon: PlusCircle, color: 'text-sky-500' },
  { key: 'pass' as const, label: 'Asignar Bono', desc: 'Crear bono para un cliente', icon: Ticket, color: 'text-pink-500' },
  { key: 'dog' as const, label: 'Registrar Perro', desc: 'Dar de alta un nuevo perro', icon: Dog, color: 'text-emerald-500' },
  { key: 'client' as const, label: 'Alta Cliente', desc: 'Invitar a un nuevo cliente', icon: UserPlus, color: 'text-amber-500' },
] as const;

export function QuickActions({ onAction }: QuickActionsProps) {
  return (
    <div>
      <h3 className="mb-3 text-sm font-semibold">Acciones rápidas</h3>
      <div className="grid grid-cols-2 gap-2">
        {actions.map(({ key, label, desc, icon: Icon, color }) => (
          <button
            key={key}
            onClick={() => onAction(key)}
            className="flex flex-col items-start gap-1 rounded-xl border border-border p-3 text-left transition-all hover:border-primary/40 hover:bg-muted/30 hover:shadow-sm"
          >
            <Icon className={`h-5 w-5 ${color}`} strokeWidth={1.5} />
            <span className="text-xs font-semibold">{label}</span>
            <span className="text-[10px] leading-tight text-muted-foreground">{desc}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

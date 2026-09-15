import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Download, FileText, X } from 'lucide-react';
import { fetchAttendanceReport, downloadAttendanceReportCSV } from '@/infrastructure/repositories/reservation-repository.impl';
import { LoadingSpinner } from '@/components/shared/loading-spinner';
import { useToast } from '@/features/ui/hooks/toast-context';
import { cn } from '@/lib/utils';
import type { ApiError } from '@/infrastructure/api/http-client';

type DateStr = string | null;

function parseErrorBody(err: unknown): string {
  const e = err as ApiError | null;
  if (!e) return 'Error desconocido';
  if (e.status === 401) return 'Sesión caducada';
  const body = e.body as { error?: string; details?: string } | null;
  if (!body) return 'Error inesperado';
  if (body.error === 'validation') return body.details || 'Valor inválido';
  return body.details || body.error || 'Error inesperado';
}

// Convert an <input type="date"> value ("YYYY-MM-DD" or "") into the
// RFC3339 form expected by the backend. `to` extends to the end of
// the day so that activities occurring later in the day are included
// (the backend treats `to` as inclusive).
function dateInputToRFC3339(date: DateStr, endOfDay: boolean): string {
  if (!date) return '';
  return endOfDay ? `${date}T23:59:59.999Z` : `${date}T00:00:00.000Z`;
}

function formatActivityDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString('es-ES', {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

export function AttendanceReportPage() {
  const toast = useToast();
  const [from, setFrom] = useState<DateStr>(null);
  const [to, setTo] = useState<DateStr>(null);
  const [downloadState, setDownloadState] = useState<'idle' | 'loading'>('idle');

  const params = useMemo(() => ({
    from: dateInputToRFC3339(from, false),
    to: dateInputToRFC3339(to, true),
  }), [from, to]);

  const { data: entries = [], isLoading, isError, error } = useQuery({
    queryKey: ['attendance', params.from, params.to],
    queryFn: () => fetchAttendanceReport(params.from, params.to),
  });

  const handleDownload = async () => {
    setDownloadState('loading');
    try {
      await downloadAttendanceReportCSV(params.from, params.to);
    } catch (err) {
      toast.error('No se pudo descargar el CSV', parseErrorBody(err));
    } finally {
      setDownloadState('idle');
    }
  };

  const handleClear = () => {
    setFrom(null);
    setTo(null);
  };

  const hasFilters = Boolean(from || to);

  return (
    <div className="px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Asistencia</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Perros que han participado (clase completada) en actividades
          </p>
        </div>
        <button
          type="button"
          onClick={handleDownload}
          disabled={downloadState === 'loading'}
          className={cn(
            'inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow hover:bg-primary/90 disabled:opacity-50',
          )}
        >
          {downloadState === 'loading' ? (
            <LoadingSpinner size="sm" />
          ) : (
            <Download className="h-4 w-4" />
          )}
          Exportar CSV
        </button>
      </div>

      <div className="mb-4 rounded-xl border border-border bg-card p-4">
        <div className="grid gap-4 sm:grid-cols-[1fr_1fr_auto_auto] sm:items-end">
          <div>
            <label htmlFor="attendance-from" className="mb-1 block text-xs font-medium text-muted-foreground">
              Desde
            </label>
            <input
              id="attendance-from"
              type="date"
              value={from ?? ''}
              max={to ?? undefined}
              onChange={(e) => setFrom(e.target.value || null)}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div>
            <label htmlFor="attendance-to" className="mb-1 block text-xs font-medium text-muted-foreground">
              Hasta
            </label>
            <input
              id="attendance-to"
              type="date"
              value={to ?? ''}
              min={from ?? undefined}
              onChange={(e) => setTo(e.target.value || null)}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          {hasFilters && (
            <button
              type="button"
              onClick={handleClear}
              className="inline-flex items-center justify-center gap-1 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:text-foreground"
            >
              <X className="h-3 w-3" />
              Limpiar
            </button>
          )}
          <p className="text-xs text-muted-foreground sm:pl-2">
            {hasFilters
              ? 'Los filtros se aplican a la fecha de la actividad.'
              : 'Sin filtros: muestra todas las reservas completadas.'}
          </p>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <LoadingSpinner size="lg" />
        </div>
      ) : isError ? (
        <div className="flex flex-col items-center justify-center gap-2 py-20">
          <p className="text-sm text-destructive">Error al cargar el reporte</p>
          <p className="text-xs text-muted-foreground">{parseErrorBody(error)}</p>
        </div>
      ) : entries.length === 0 ? (
        <EmptyState hasFilters={hasFilters} onClear={handleClear} />
      ) : (
        <AttendanceTable entries={entries} count={entries.length} />
      )}
    </div>
  );
}

function EmptyState({ hasFilters, onClear }: { hasFilters: boolean; onClear: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-border bg-card py-20 text-center">
      <FileText className="h-10 w-10 text-muted-foreground" />
      <p className="text-sm font-medium">No hay datos de asistencia</p>
      {hasFilters ? (
        <>
          <p className="max-w-md text-sm text-muted-foreground">
            No se encontraron reservas completadas en el rango seleccionado.
          </p>
          <button
            type="button"
            onClick={onClear}
            className="text-xs font-medium text-primary hover:underline"
          >
            Quitar filtros
          </button>
        </>
      ) : (
        <p className="text-sm text-muted-foreground">
          Cuando un perro complete una actividad aparecerá aquí.
        </p>
      )}
    </div>
  );
}

function AttendanceTable({ entries, count }: { entries: import('@/domain/entities/attendance-report').AttendanceReportEntry[]; count: number }) {
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <span className="text-xs font-medium text-muted-foreground">
          {count} {count === 1 ? 'asistencia registrada' : 'asistencias registradas'}
        </span>
      </div>
      <div className="hidden md:block">
        <table className="w-full">
          <thead>
            <tr className="border-b border-border bg-muted/30 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3">Perro</th>
              <th className="px-4 py-3">Pasaporte</th>
              <th className="px-4 py-3">Actividad</th>
              <th className="px-4 py-3">Fecha</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {entries.map((e) => (
              <tr key={e.reservation_id} className="hover:bg-muted/30">
                <td className="px-4 py-3 text-sm font-medium">{e.dog_name}</td>
                <td className="px-4 py-3 text-sm text-muted-foreground tabular-nums">{e.dog_passport}</td>
                <td className="px-4 py-3 text-sm">{e.activity_name}</td>
                <td className="px-4 py-3 text-sm tabular-nums text-muted-foreground">
                  {formatActivityDate(e.activity_date)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="space-y-2 p-4 md:hidden">
        {entries.map((e) => (
          <div key={e.reservation_id} className="rounded-lg border border-border p-3">
            <div className="flex items-baseline justify-between gap-2">
              <p className="font-medium">{e.dog_name}</p>
              <p className="text-xs tabular-nums text-muted-foreground">{formatActivityDate(e.activity_date)}</p>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              <span className="font-medium">Pasaporte:</span> {e.dog_passport}
            </p>
            <p className="mt-1 text-sm">{e.activity_name}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

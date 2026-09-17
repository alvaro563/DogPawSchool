import type { ApiError } from '@/infrastructure/api/http-client';

interface ErrorBody {
  error?: string;
  field?: string;
  details?: string;
}

/**
 * Maps a reservation API error to a Spanish message suitable for inline
 * display or toast bodies. The function inspects the body's `error`
 * field (the wire-level code emitted by the backend) and falls back to
 * the HTTP status for codes that are not enumerated here. If the body
 * carries a `details` string the server already produced the user-facing
 * explanation; we use it as-is when present.
 */
export function getReservationErrorMessage(err: ApiError, fallback: string): string {
  const body = err?.body as ErrorBody | null | undefined;
  const code = body?.error;
  const details = body?.details;

  switch (code) {
    case 'sex_neutered_incompatibility':
      return details ?? 'Hay incompatibilidad de sexo/castración con otros perros inscritos.';
    case 'dog_incompatible':
      return details ?? 'Hay incompatibilidades con otros perros inscritos.';
    case 'dog_size_mismatch':
      return details ?? 'El tamaño del perro no coincide con el de la actividad.';
    case 'activity_full':
      return 'No hay plazas disponibles en esta actividad.';
    case 'duplicate_reservation':
      return 'Este perro ya tiene una reserva en esta actividad.';
    case 'activity_in_past':
      return 'Esta actividad ya ha pasado.';
    case 'activity_not_started':
      return 'La actividad aún no ha empezado.';
    case 'activity_not_finished':
      return 'La actividad aún no ha terminado.';
    case 'not_cancellable':
      return 'La reserva no se puede cancelar en su estado actual.';
    case 'not_completable':
      return 'La reserva no se puede completar en su estado actual.';
    case 'not_pending':
      return 'La reserva no está pendiente de confirmación.';
    case 'not_late_cancelled':
      return 'Esta reserva ya no está en estado "cancelada tarde" (puede que ya haya sido perdonada por otro administrador).';
    case 'already_cancelled':
      return 'La reserva ya estaba cancelada.';
    case 'invalid_activity_id':
    case 'invalid_dog_id':
    case 'invalid_pass_id':
      return details ?? 'Faltan datos para crear la reserva.';
    case 'pass_exhausted':
      return 'El bono no tiene sesiones disponibles.';
    case 'pass_expired':
      return 'El bono ha caducado.';
    case 'dog_pass_owner_mismatch':
      return 'El perro y el bono deben pertenecer al mismo usuario.';
    case 'validation':
      if (body?.field) return `Error en ${body.field}: ${details ?? 'valor inválido'}`;
      return details ?? 'Datos inválidos.';
  }

  if (err?.status === 409) return 'Conflicto con la reserva.';
  if (err?.status === 400) return details ?? 'Datos inválidos.';
  if (err?.status === 404) return 'Recurso no encontrado.';
  return fallback;
}

/**
 * Builds the description string for the pending-to-confirm toast. When
 * the backend provides explicit reasons (an array of Spanish strings),
 * they are joined with newlines so the user can read each one. When the
 * array is missing, a generic fallback message is returned.
 */
export function pendingReasonsDescription(
  reasons: string[] | undefined | null,
  dogName: string,
): string {
  if (reasons && reasons.length > 0) return reasons.join('\n');
  return `${dogName} queda en lista de espera por incompatibilidades. La escuela revisará la reserva y te avisaremos.`;
}

package handler

import (
	"fmt"
	"strings"

	"dogpaw/internal/domain"
	"dogpaw/internal/usecase/reservation"
)

// formatPendingReasons translates the use case's PendingReason codes
// (stable, language-neutral) into Spanish strings for the wire
// response. When dogNames is provided, dog IDs are resolved to names;
// otherwise the bare ID is used as a fallback.
//
// The function never returns nil when the input is non-nil; it returns
// an empty slice for an empty input. Stable ordering is preserved: the
// output slice is in the same order as the input.
func formatPendingReasons(reasons []reservation.PendingReason, dogNames map[int]string) []string {
	if len(reasons) == 0 {
		return nil
	}
	out := make([]string, 0, len(reasons))
	for _, r := range reasons {
		out = append(out, formatPendingReason(r, dogNames))
	}
	return out
}

// formatPendingReason produces a single user-facing Spanish sentence
// explaining why a reservation is pending.
func formatPendingReason(r reservation.PendingReason, dogNames map[int]string) string {
	code := strings.TrimPrefix(r.Code, reservation.SexNeuteredReasonPrefix)
	switch code {
	case domain.ReasonHasSpecialCondition:
		name := nameOrID(r.DogIDs, dogNames, 0)
		return fmt.Sprintf("%s tiene una condición especial y requiere revisión por la escuela.", name)
	case domain.ReasonCastratedVsCastrated:
		incoming, existing := pairNames(r.DogIDs, dogNames)
		return fmt.Sprintf("%s queda pendiente: coincide con %s, ambos son machos castrados.", incoming, existing)
	case domain.ReasonIntactVsCastrated:
		incoming, existing := pairNames(r.DogIDs, dogNames)
		return fmt.Sprintf("%s queda pendiente: coincide con %s (macho castrado).", incoming, existing)
	case domain.ReasonCastratedVsIntact:
		incoming, existing := pairNames(r.DogIDs, dogNames)
		return fmt.Sprintf("%s queda pendiente: coincide con %s (macho sin castrar).", incoming, existing)
	case domain.ReasonIntactVsIntact:
		// Pending-but-blocker is also possible when the caller chose to
		// surface this code in a non-blocking context. Use the same
		// wording the blocker path uses, since the underlying issue is
		// the same.
		incoming, existing := pairNames(r.DogIDs, dogNames)
		return fmt.Sprintf("%s queda pendiente: coincide con %s, ambos son machos sin castrar.", incoming, existing)
	}
	// Unknown code: surface as-is so the frontend can show something
	// meaningful even when a code is added server-side before the
	// frontend is updated.
	return fmt.Sprintf("Reserva pendiente (motivo: %s).", r.Code)
}

// pairNames returns (incomingName, existingName) for a two-ID DogIDs
// slice. When dogNames is nil or a name is missing, the ID itself is
// used as a fallback.
func pairNames(dogIDs []int, dogNames map[int]string) (string, string) {
	incoming := nameOrID(dogIDs, dogNames, 0)
	existing := nameOrID(dogIDs, dogNames, 1)
	return incoming, existing
}

// nameOrID picks the index-th dog ID from dogIDs and resolves it via
// dogNames, falling back to the bare ID as "#N" when no name is found.
// Returns "el perro" if the index is out of range.
func nameOrID(dogIDs []int, dogNames map[int]string, index int) string {
	if index < 0 || index >= len(dogIDs) {
		return "el perro"
	}
	id := dogIDs[index]
	if dogNames != nil {
		if n, ok := dogNames[id]; ok && n != "" {
			return n
		}
	}
	return fmt.Sprintf("el perro #%d", id)
}

// formatSexNeuteredBlockingError produces the user-facing Spanish
// message returned to clients when a reservation is rejected because
// two intact males would end up in the same activity. The output is
// a single human-readable sentence aggregating every blocker pair.
func formatSexNeuteredBlockingError(err *reservation.SexNeuteredConflictError) string {
	if err == nil || err.IncomingDog == nil {
		return "No se puede apuntar: hay incompatibilidad de sexo/castración."
	}
	names := make([]string, 0, len(err.BlockingDogs))
	for _, d := range err.BlockingDogs {
		names = append(names, d.Name())
	}
	if len(names) == 0 {
		return fmt.Sprintf("No se puede apuntar a %s: coincide con uno o más machos sin castrar en la actividad.", err.IncomingDog.Name())
	}
	if len(names) == 1 {
		return fmt.Sprintf("No se puede apuntar a %s: coincide con %s (macho sin castrar).", err.IncomingDog.Name(), names[0])
	}
	return fmt.Sprintf("No se puede apuntar a %s: coincide con %s (machos sin castrar).",
		err.IncomingDog.Name(), joinNames(names))
}

// joinNames joins a small list of names with commas and "y" before
// the last element (Spanish-style: "A, B y C").
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " y " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + " y " + names[len(names)-1]
}

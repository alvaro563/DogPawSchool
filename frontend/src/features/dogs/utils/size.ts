export type SizeBracket = 'MINI' | 'MEDIUM' | 'LARGE' | 'UNKNOWN';

export const sizeBracketLabel: Record<SizeBracket, string> = {
  MINI: 'Mini',
  MEDIUM: 'Mediano',
  LARGE: 'Grande',
  UNKNOWN: '—',
};

// getSizeBracket mirrors the backend's domain.Dog.SizeBracket()
// derivation from weightKg. Thresholds must stay in sync with
// internal/domain/dog.go (WeightMiniMaxKg = 5, WeightMediumMaxKg = 20).
export function getSizeBracket(weightKg: number): SizeBracket {
  if (weightKg <= 0) return 'UNKNOWN';
  if (weightKg <= 5) return 'MINI';
  if (weightKg <= 20) return 'MEDIUM';
  return 'LARGE';
}

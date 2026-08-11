import { cn } from '@/lib/utils';

const SEX_LABELS: Record<string, string> = { MALE: '♂ Macho', FEMALE: '♀ Hembra' };

interface SexChipProps {
  sex: string;
  showIcon?: boolean;
}

export function SexChip({ sex, showIcon = false }: SexChipProps) {
  const isMale = sex === 'MALE';

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium',
        isMale
          ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
          : 'bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-300',
      )}
    >
      {showIcon && <span className="text-[10px]">{isMale ? '♂' : '♀'}</span>}
      {SEX_LABELS[sex] || sex}
    </span>
  );
}

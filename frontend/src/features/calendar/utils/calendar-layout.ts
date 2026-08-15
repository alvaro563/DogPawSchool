import type { Activity } from '@/domain/entities/activity';

export interface PositionedActivity {
  activity: Activity;
  startMinutes: number;
  endMinutes: number;
  column: number;
  totalColumns: number;
}

/** Assigns parallel columns to overlapping activity intervals. */
export function layoutOverlappingActivities(
  activities: Activity[],
): PositionedActivity[] {
  const sorted = activities
    .map((activity) => {
      const start = new Date(activity.date);
      const startMinutes = start.getHours() * 60 + start.getMinutes();
      return {
        activity,
        startMinutes,
        endMinutes: startMinutes + activity.duration_in_hours * 60,
      };
    })
    .sort((a, b) => a.startMinutes - b.startMinutes || a.endMinutes - b.endMinutes);

  const result: PositionedActivity[] = [];
  let cluster: typeof sorted = [];
  let clusterEnd = -Infinity;

  const flush = () => {
    if (cluster.length === 0) return;
    const columns: number[] = [];
    const assignments = cluster.map((item) => {
      let column = columns.findIndex((end) => end <= item.startMinutes);
      if (column < 0) {
        column = columns.length;
        columns.push(item.endMinutes);
      } else {
        columns[column] = item.endMinutes;
      }
      return { ...item, column };
    });
    const totalColumns = columns.length;
    result.push(...assignments.map((item) => ({ ...item, totalColumns })));
    cluster = [];
    clusterEnd = -Infinity;
  };

  for (const item of sorted) {
    if (cluster.length > 0 && item.startMinutes >= clusterEnd) flush();
    cluster.push(item);
    clusterEnd = Math.max(clusterEnd, item.endMinutes);
  }
  flush();

  return result;
}

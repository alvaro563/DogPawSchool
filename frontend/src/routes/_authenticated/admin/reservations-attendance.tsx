import { createFileRoute } from '@tanstack/react-router';
import { AttendanceReportPage } from '@/features/admin/pages/attendance-report-page';

export const Route = createFileRoute('/_authenticated/admin/reservations-attendance')({
  component: AttendanceReportPage,
});

export interface AttendanceReportEntry {
  reservation_id: number;
  activity_id: number;
  activity_name: string;
  activity_date: string;
  dog_id: number;
  dog_name: string;
  dog_passport: string;
}

export interface AttendanceReportResponse {
  entries: AttendanceReportEntry[];
  limit: number;
  offset: number;
  count: number;
}

export function formatRelativeTimestamp(timestamp?: string): string {
  if (!timestamp) return '';
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const diffInMs = now.getTime() - date.getTime();
    const diffInDays = Math.floor(diffInMs / (1000 * 60 * 60 * 24));
    const diffInHours = Math.floor(diffInMs / (1000 * 60 * 60));
    const diffInMinutes = Math.floor(diffInMs / (1000 * 60));

    if (diffInDays > 0) return `${diffInDays}d ago`;
    if (diffInHours > 0) return `${diffInHours}h ago`;
    if (diffInMinutes > 0) return `${diffInMinutes}m ago`;
    return 'Just now';
  } catch {
    return '';
  }
}

export function formatFullTimestamp(
  timestamp?: string,
  timezone?: string | null
): string {
  if (!timestamp) return '';
  try {
    const date = new Date(timestamp);
    const userTimezone =
      timezone || Intl.DateTimeFormat().resolvedOptions().timeZone;
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      timeZone: userTimezone,
      timeZoneName: 'short',
    }).format(date);
  } catch {
    return '';
  }
}

// Formats a Date as YYYY-MM-DD using the user's local timezone, avoiding UTC rollover issues.
export function formatDateLocal(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

// Formats a Date or date-like string into YYYY-MM-DD in local time.
export function formatDateForAPI(date: Date | string): string {
  if (typeof date === 'string') {
    if (/^\d{4}-\d{2}-\d{2}$/.test(date)) return date;
    return formatDateLocal(new Date(date));
  }
  return formatDateLocal(date);
}

// Parses a YYYY-MM-DD string into a Date constructed in the user's local timezone.
// This avoids interpreting the string as UTC (which can shift the date by one day
// in timezones west of UTC when appending a time component like T00:00:00).
export function parseLocalDateString(dateString: string): Date | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(dateString);
  if (!match) return null;
  const [, yearStr, monthStr, dayStr] = match;
  const year = Number(yearStr);
  const monthIndex = Number(monthStr) - 1; // JS Date months are 0-based
  const day = Number(dayStr);
  const date = new Date(year, monthIndex, day);
  if (isNaN(date.getTime())) return null;
  return date;
}

// Formats a date string into the format "Jan 2, 2025"
export function formatDateCreated(timestamp?: string): string {
  if (!timestamp) return 'N/A';
  try {
    const date = new Date(timestamp);
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    }).format(date);
  } catch {
    return 'N/A';
  }
}

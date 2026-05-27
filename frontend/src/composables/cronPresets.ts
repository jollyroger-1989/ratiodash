export const CRON_PRESETS = [
  { value: '*/15 * * * *', label: 'trackers.modal.schedule15min' },
  { value: '*/30 * * * *', label: 'trackers.modal.schedule30min' },
  { value: '@hourly',      label: 'trackers.modal.scheduleHourly' },
  { value: '0 */2 * * *', label: 'trackers.modal.schedule2h' },
  { value: '0 */6 * * *', label: 'trackers.modal.schedule6h' },
  { value: '0 */12 * * *', label: 'trackers.modal.schedule12h' },
  { value: '@daily',       label: 'trackers.modal.scheduleDaily' },
  { value: '@weekly',      label: 'trackers.modal.scheduleWeekly' },
  { value: '@monthly',     label: 'trackers.modal.scheduleMonthly' },
] as const

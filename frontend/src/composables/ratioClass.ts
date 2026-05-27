export function ratioClass(r: number): string {
  if (r >= 1) return 'ratio-good'
  if (r >= 0.5) return 'ratio-warn'
  return 'ratio-bad'
}

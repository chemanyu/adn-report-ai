// Round decimal strings without losing precision through Number conversion.
export function formatDecimal(value, { grouping = false } = {}) {
  const text = String(value ?? '')
  const match = /^([+-]?)(\d+)(?:\.(\d*))?$/.exec(text)
  if (!match) return text

  const [, sign, integer, fraction = ''] = match
  let cents = BigInt(integer) * 100n + BigInt(fraction.padEnd(2, '0').slice(0, 2))
  if (fraction[2] >= '5') cents += 1n

  let whole = String(cents / 100n)
  if (grouping) whole = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  const decimal = String(cents % 100n).padStart(2, '0')
  return `${sign === '-' && cents !== 0n ? '-' : ''}${whole}.${decimal}`
}

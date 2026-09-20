// Client-side validation mirroring the backend rules, so the user gets fast
// feedback and we don't fire obviously-invalid requests.

// Stellar public key: 'G' + 55 base32 chars (A-Z, 2-7) = 56 chars total.
export const isStellarPubKey = (s: string): boolean => /^G[A-Z2-7]{55}$/.test(s)

// Stellar amounts: positive decimal string, <= 7 decimal places,
// <= 922337203685.4775807 (int64 stroops).
const MAX_AMOUNT = 922337203685.4775807

export function validateAmount(s: string): string | null {
  const v = s.trim()
  if (!v) return 'Amount is required'
  if (!/^\d+(\.\d+)?$/.test(v)) return 'Must be a positive decimal number'
  const dec = v.split('.')[1] ?? ''
  if (dec.length > 7) return 'At most 7 decimal places'
  const n = Number(v)
  if (!(n > 0)) return 'Must be greater than 0'
  if (n > MAX_AMOUNT) return 'Amount is too large'
  return null
}

export function validateProductRef(s: string): string | null {
  const v = s.trim()
  if (!v) return 'Product reference is required'
  if (v.length > 255) return 'At most 255 characters'
  return null
}

// Amounts are compared as stroops (integer 1e-7 units) rather than as doubles:
// the top of the Stellar range, 922337203685.4775807, does not survive a float
// round-trip, and this guards a payment amount.
export const stroops = (s: string): bigint => {
  const [whole, frac = ''] = s.trim().split('.')
  return BigInt(whole + frac.padEnd(7, '0').slice(0, 7))
}

// Soft check for the new-order form: an order larger than the wallet holds is
// still a valid order — the wallet can be funded before it is paid — so this
// warns rather than blocks. Callers must pass an amount validateAmount accepted.
export function exceedsBalance(amount: string, balance: string): boolean {
  return stroops(amount) > stroops(balance)
}

// Back to a Stellar amount string. Kept beside stroops() so every decimal
// conversion in the app lives in one tested place.
export const toAmount = (v: bigint): string => {
  const s = v.toString().padStart(8, '0')
  return `${s.slice(0, -7)}.${s.slice(-7)}`
}

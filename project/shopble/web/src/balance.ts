// Buyer-side balance lookup. Plain Horizon REST rather than stellar-sdk: this runs
// on the new-order form, and pulling the ~400 kB sdk into the initial bundle for a
// single GET would undo the dynamic import that keeps it out of pay.ts.
import { HORIZON_URL } from './pay'
import { stroops, toAmount } from './validation'

export type Balance =
  | { kind: 'ok'; amount: string }
  | { kind: 'no_trustline' }
  | { kind: 'no_account' }

interface HorizonBalance {
  asset_code?: string
  asset_issuer?: string
  balance: string
  selling_liabilities?: string
}

// The order asset always has an issuer — the backend refuses a config where
// asset_code or asset_issuer is empty — so this never looks at the native balance
// and owes no base-reserve arithmetic. Transaction fees are paid in XLM, not here.
export async function assetBalance(
  account: string,
  code: string,
  issuer: string,
): Promise<Balance> {
  const res = await fetch(`${HORIZON_URL}/accounts/${encodeURIComponent(account)}`)
  if (res.status === 404) return { kind: 'no_account' }
  if (!res.ok) throw new Error(`Horizon returned HTTP ${res.status}`)
  const { balances } = (await res.json()) as { balances: HorizonBalance[] }
  const held = balances.find((b) => b.asset_code === code && b.asset_issuer === issuer)
  // Absent from the list means no trustline, which is different from holding zero:
  // without the line the payment cannot be sent at all.
  if (!held) return { kind: 'no_trustline' }
  // Balance is not the same as spendable: anything reserved by open DEX offers
  // cannot be sent, so a Max built from the raw balance would be unpayable.
  const free = stroops(held.balance) - stroops(held.selling_liabilities ?? '0')
  return { kind: 'ok', amount: toAmount(free > 0n ? free : 0n) }
}

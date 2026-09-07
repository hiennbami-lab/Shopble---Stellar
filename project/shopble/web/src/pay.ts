// Deliverable 1's other half: the payment instruction as a signable transaction,
// not just a copyable payload. Builds the payment locally, hands the XDR to the
// connected wallet for signature, then submits it to Horizon.
import { StellarWalletsKit } from '@creit.tech/stellar-wallets-kit'
import { TESTNET_PASSPHRASE } from './api'

const HORIZON_URL = import.meta.env.VITE_HORIZON ?? 'https://horizon-testnet.stellar.org'

export interface PayFields {
  destination: string
  asset_code: string
  asset_issuer: string
  amount: string
  memo: string
}

export type PayPhase = 'building' | 'signing' | 'submitting'

export async function payWithWallet(
  source: string,
  fields: PayFields,
  onPhase?: (phase: PayPhase) => void,
): Promise<string> {
  // stellar-sdk is ~400 kB and only needed once someone actually pays, so it is
  // loaded on demand rather than in the initial bundle.
  const { Asset, BASE_FEE, Horizon, Memo, Operation, TransactionBuilder } =
    await import('@stellar/stellar-sdk')
  const server = new Horizon.Server(HORIZON_URL)

  onPhase?.('building')
  const account = await server.loadAccount(source)
  // An empty issuer means native XLM; anything else is a credit asset.
  const asset = fields.asset_issuer
    ? new Asset(fields.asset_code, fields.asset_issuer)
    : Asset.native()

  const tx = new TransactionBuilder(account, {
    fee: BASE_FEE,
    networkPassphrase: TESTNET_PASSPHRASE,
  })
    .addOperation(
      Operation.payment({ destination: fields.destination, asset, amount: fields.amount }),
    )
    // The memo is the order id. Reconciliation cannot match the payment without it.
    .addMemo(Memo.text(fields.memo))
    .setTimeout(180)
    .build()

  onPhase?.('signing')
  const { signedTxXdr } = await StellarWalletsKit.signTransaction(tx.toXDR(), {
    networkPassphrase: TESTNET_PASSPHRASE,
    address: source,
  })

  onPhase?.('submitting')
  const signed = TransactionBuilder.fromXDR(signedTxXdr, TESTNET_PASSPHRASE)
  const res = await server.submitTransaction(signed)
  return res.hash
}

// Horizon reports failures as result codes buried in the error response. Turn the
// ones a buyer can act on into plain language, and never swallow the raw code.
const CODE_TEXT: Record<string, string> = {
  op_no_trust: 'The destination account cannot hold this asset yet — it has no trustline for it.',
  op_underfunded: 'This wallet does not hold enough of the asset to cover the payment.',
  op_no_destination: 'The destination account does not exist on this network.',
  op_line_full: 'The destination account cannot receive any more of this asset.',
  tx_insufficient_fee: 'The network fee was too low. Try again.',
  tx_bad_seq: 'The wallet used a stale sequence number. Try again.',
  tx_too_late: 'The transaction expired before it was submitted. Try again.',
}

export function payErrorText(e: unknown): string {
  const codes = (e as { response?: { data?: { extras?: { result_codes?: {
    transaction?: string
    operations?: string[]
  } } } } })?.response?.data?.extras?.result_codes

  if (codes) {
    const code = codes.operations?.find((c) => c !== 'op_success') ?? codes.transaction
    if (code) return CODE_TEXT[code] ? `${CODE_TEXT[code]} (${code})` : code
  }
  if (e instanceof Error) return e.message
  return 'The payment could not be submitted.'
}

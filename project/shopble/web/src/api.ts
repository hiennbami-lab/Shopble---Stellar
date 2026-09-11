// Typed client for the Shopble backend. All responses are enveloped:
//   success: { status: "ok", data: ... }
//   error:   { status: "error", data: { error_code, message } }

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'

export type OrderStatus =
  | 'awaiting_payment'
  | 'payment_detected'
  | 'validated'
  | 'rejected'

export type RejectReason =
  | ''
  | 'underpayment'
  | 'wrong_asset'
  | 'wrong_destination'
  | 'wrong_buyer_wallet'
  | 'invalid_memo'

export interface OrderDto {
  id: string
  product_ref: string
  expected_amount: string
  asset_code: string
  asset_issuer: string
  destination_account: string
  memo: string
  buyer_wallet: string
  status: OrderStatus
  reject_reason?: RejectReason
  on_chain_tx_hash?: string
  created_at: number
  updated_at: number
}

export const POLL_MS = 4000

// How long to keep polling a settled order for its on-chain verdict.
// The watcher allows chainSyncMaxAttempts = 3 attempts, each capped at a 3-minute
// context timeout, with sweeps ~3s apart: 3*180s + 2*3s = 546s worst case. The budget
// has to outlive that or we would give up mid-retry and claim nothing was written.
export const CHAIN_WAIT_TICKS = 140

// Terminal states: the matcher has judged the payment and will not change its mind.
export const isSettled = (o: Pick<OrderDto, 'status'>) =>
  o.status === 'validated' || o.status === 'rejected'

export interface PaymentInstruction {
  destination: string
  asset_code: string
  asset_issuer: string
  amount: string
  memo: string
  memo_type: string
  network: string
  uri: string
}

export interface CreateOrderResult {
  order: OrderDto
  instruction: PaymentInstruction
}

export interface CreateOrderBody {
  product_ref: string
  expected_amount: string
  buyer_wallet: string
}

export class ApiError extends Error {
  code: string
  constructor(code: string, message: string) {
    super(message)
    this.code = code
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response
  try {
    res = await fetch(`${API_BASE}${path}`, {
      headers: { 'Content-Type': 'application/json' },
      ...init,
    })
  } catch {
    throw new ApiError('NETWORK', `Cannot reach backend at ${API_BASE}`)
  }
  let body: { status?: string; data?: unknown }
  try {
    body = await res.json()
  } catch {
    throw new ApiError('BAD_RESPONSE', `Non-JSON response (HTTP ${res.status})`)
  }
  if (body?.status !== 'ok') {
    const d = (body?.data ?? {}) as { error_code?: string; message?: string }
    throw new ApiError(d.error_code ?? 'UNKNOWN', d.message ?? `Request failed (HTTP ${res.status})`)
  }
  return body.data as T
}

export const createOrder = (b: CreateOrderBody) =>
  request<CreateOrderResult>('/api/v1/orders', {
    method: 'POST',
    body: JSON.stringify(b),
  })

export const listOrders = (buyerWallet: string, limit = 50) =>
  request<OrderDto[]>(
    `/api/v1/orders?buyer_wallet=${encodeURIComponent(buyerWallet)}&limit=${limit}`,
  )

export const TESTNET_PASSPHRASE = 'Test SDF Network ; September 2015'

// EvidenceRecord carries a ready-made `explorer_url`; the order-level hash does not,
// so build it here rather than inlining the host at each call site.
export const explorerTx = (hash: string) =>
  `https://stellar.expert/explorer/testnet/tx/${hash}`

// The backend only returns `instruction.uri` when the order is created. An order
// reopened from history has every field the SEP-0007 pay URI needs, so rebuild it
// rather than hiding the QR.
export function payUri(o: OrderDto): string {
  const q = new URLSearchParams({
    destination: o.destination_account,
    amount: o.expected_amount,
    asset_code: o.asset_code,
    asset_issuer: o.asset_issuer,
    memo: o.memo,
    memo_type: 'MEMO_TEXT',
    network_passphrase: TESTNET_PASSPHRASE,
  })
  return `web+stellar:pay?${q}`
}

export type Verdict = 'matched' | 'rejected' | 'duplicate'

export interface EvidenceRecord {
  id: string
  op_id: string
  tx_hash: string
  explorer_url: string
  source_account: string
  destination_account: string
  asset_code: string
  asset_issuer: string
  amount: string
  memo: string
  ledger_close_at: number
  captured_at: number
  detection_latency_seconds: number
  order_id?: string
  verdict: Verdict
  reject_reason?: RejectReason
}

export interface ComparisonRow {
  field: string
  expected: string
  observed: string
  match: boolean
}

export interface OrderEvidence {
  evidence: EvidenceRecord
  comparison: ComparisonRow[]
}

export interface OrderEvidenceResult {
  order: OrderDto
  evidence: OrderEvidence[]
}

// Returns the order *and* everything observed for it, so the order screen needs
// one request rather than two.
export const getOrderEvidence = (id: string) =>
  request<OrderEvidenceResult>(`/api/v1/orders/${encodeURIComponent(id)}/evidence`)

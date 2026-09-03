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
  reject_reason: RejectReason
  on_chain_tx_hash: string
  created_at: number
  updated_at: number
}

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

export const getOrder = (id: string) =>
  request<OrderDto>(`/api/v1/orders/${encodeURIComponent(id)}`)

export const listOrders = (buyerWallet: string, limit = 50) =>
  request<OrderDto[]>(
    `/api/v1/orders?buyer_wallet=${encodeURIComponent(buyerWallet)}&limit=${limit}`,
  )

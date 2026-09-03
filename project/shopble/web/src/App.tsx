import { useCallback, useEffect, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import {
  ApiError,
  createOrder,
  getOrder,
  listOrders,
  type CreateOrderResult,
  type OrderDto,
  type OrderStatus,
  type RejectReason,
} from './api'
import { connectWallet, disconnectWallet } from './wallet'
import { isStellarPubKey, validateAmount, validateProductRef } from './validation'

// ---- Status / reject-reason metadata (first-class in the UI even though the
// backend can't flip past awaiting_payment yet — Deliverable 2/3 not built). ----

const STATUS_META: Record<OrderStatus, { label: string; cls: string }> = {
  awaiting_payment: { label: 'Awaiting payment', cls: 'badge-wait' },
  payment_detected: { label: 'Payment detected', cls: 'badge-detected' },
  validated: { label: 'Validated', cls: 'badge-ok' },
  rejected: { label: 'Rejected', cls: 'badge-bad' },
}

const REJECT_LABEL: Record<Exclude<RejectReason, ''>, string> = {
  underpayment: 'Underpayment',
  wrong_asset: 'Wrong asset',
  wrong_destination: 'Wrong destination',
  wrong_buyer_wallet: 'Wrong buyer wallet',
  invalid_memo: 'Invalid memo',
}

const shortKey = (k: string) => (k.length > 12 ? `${k.slice(0, 6)}…${k.slice(-6)}` : k)

// ---------------------------------------------------------------------------

type Tab = 'new' | 'history'

export default function App() {
  const [wallet, setWallet] = useState<string | null>(null)
  const [tab, setTab] = useState<Tab>('new')
  const [trackingId, setTrackingId] = useState<string | null>(null)

  const onConnect = async () => {
    try {
      setWallet(await connectWallet())
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Wallet connection failed')
    }
  }

  return (
    <div className="app">
      <header>
        <h1>Shopble</h1>
        <span className="net">Stellar Testnet</span>
        <div className="spacer" />
        {wallet ? (
          <div className="wallet">
            <code title={wallet}>{shortKey(wallet)}</code>
            <button className="ghost" onClick={() => { disconnectWallet().catch(() => {}); setWallet(null) }}>Disconnect</button>
          </div>
        ) : (
          <button onClick={onConnect}>Connect wallet</button>
        )}
      </header>

      {!wallet ? (
        <p className="hint">Connect a Stellar testnet wallet (Freighter / Wallets Kit) to create an order.</p>
      ) : (
        <>
          <nav className="tabs">
            <button className={tab === 'new' ? 'active' : ''} onClick={() => setTab('new')}>New order</button>
            <button className={tab === 'history' ? 'active' : ''} onClick={() => setTab('history')}>My orders</button>
          </nav>

          {tab === 'new' && <NewOrder wallet={wallet} onTrack={(id) => setTrackingId(id)} />}
          {tab === 'history' && <History wallet={wallet} onTrack={(id) => setTrackingId(id)} />}
        </>
      )}

      {trackingId && <OrderDetail id={trackingId} onClose={() => setTrackingId(null)} />}
    </div>
  )
}

// ---- Create order + payment instruction ----------------------------------

function NewOrder({ wallet, onTrack }: { wallet: string; onTrack: (id: string) => void }) {
  const [productRef, setProductRef] = useState('')
  const [amount, setAmount] = useState('')
  const [result, setResult] = useState<CreateOrderResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    const pErr = validateProductRef(productRef)
    const aErr = validateAmount(amount)
    if (pErr || aErr) {
      setErr(pErr ?? aErr)
      return
    }
    setBusy(true)
    try {
      const res = await createOrder({
        product_ref: productRef.trim(),
        expected_amount: amount.trim(),
        buyer_wallet: wallet,
      })
      setResult(res)
    } catch (e) {
      setErr(e instanceof ApiError ? `${e.code}: ${e.message}` : 'Request failed')
    } finally {
      setBusy(false)
    }
  }

  if (result) {
    return (
      <PaymentInstruction
        res={result}
        onTrack={() => onTrack(result.order.id)}
        onNew={() => { setResult(null); setProductRef(''); setAmount('') }}
      />
    )
  }

  return (
    <form className="card" onSubmit={submit}>
      <h2>New order</h2>
      <label>
        Product reference
        <input value={productRef} onChange={(e) => setProductRef(e.target.value)} placeholder="sku-1024" />
      </label>
      <label>
        Expected amount (USDC)
        <input value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="25.5" inputMode="decimal" />
      </label>
      <p className="muted">Buyer wallet: <code>{shortKey(wallet)}</code></p>
      {err && <p className="error">{err}</p>}
      <button type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create order'}</button>
    </form>
  )
}

function PaymentInstruction({
  res,
  onTrack,
  onNew,
}: {
  res: CreateOrderResult
  onTrack: () => void
  onNew: () => void
}) {
  const { order, instruction } = res
  return (
    <div className="card">
      <h2>Pay this order</h2>
      <div className="pay">
        <div className="qr">
          <QRCodeSVG value={instruction.uri} size={200} />
          <p className="muted">Scan with a Stellar wallet</p>
        </div>
        <dl className="fields">
          <dt>Amount</dt><dd>{instruction.amount} {instruction.asset_code}</dd>
          <dt>Destination</dt><dd><code title={instruction.destination}>{shortKey(instruction.destination)}</code></dd>
          <dt>Asset issuer</dt><dd><code title={instruction.asset_issuer}>{shortKey(instruction.asset_issuer)}</code></dd>
          <dt>Memo ({instruction.memo_type})</dt>
          <dd className="memo"><code>{instruction.memo}</code></dd>
          <dt>Network</dt><dd>{instruction.network}</dd>
        </dl>
      </div>
      <p className="warn">
        The payment <strong>must</strong> carry memo <code>{instruction.memo}</code> ({instruction.memo_type})
        — it is the order id. Without it, the payment cannot be reconciled to this order.
      </p>
      <div className="row">
        {/* Opens the SEP-0007 pay URI in a registered wallet handler (e.g. Freighter). */}
        <a className="btn" href={instruction.uri}>Open in wallet</a>
        <button onClick={onTrack}>Track order</button>
        <button className="ghost" onClick={onNew}>New order</button>
      </div>
      <p className="muted">Order id: <code>{order.id}</code></p>
    </div>
  )
}

// ---- Order detail (polls status) -----------------------------------------

function OrderDetail({ id, onClose }: { id: string; onClose: () => void }) {
  const [order, setOrder] = useState<OrderDto | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      setOrder(await getOrder(id))
      setErr(null)
    } catch (e) {
      setErr(e instanceof ApiError ? `${e.code}: ${e.message}` : 'Failed to load order')
    }
  }, [id])

  useEffect(() => {
    load()
    // Poll until terminal. Backend can't flip past awaiting_payment yet, but the
    // loop is ready for payment_detected/validated/rejected once Deliverable 2 lands.
    const t = setInterval(() => {
      setOrder((o) => {
        if (o && (o.status === 'validated' || o.status === 'rejected')) return o
        load()
        return o
      })
    }, 4000)
    return () => clearInterval(t)
  }, [load])

  return (
    <div className="modal" onClick={onClose}>
      <div className="card modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="row between">
          <h2>Order</h2>
          <button className="ghost" onClick={onClose}>Close</button>
        </div>
        {err && <p className="error">{err}</p>}
        {!order ? (
          <p className="muted">Loading…</p>
        ) : (
          <>
            <StatusBadge status={order.status} reason={order.reject_reason} />
            <dl className="fields">
              <dt>Order id</dt><dd><code>{order.id}</code></dd>
              <dt>Product</dt><dd>{order.product_ref}</dd>
              <dt>Expected</dt><dd>{order.expected_amount} {order.asset_code}</dd>
              <dt>Buyer wallet</dt><dd><code title={order.buyer_wallet}>{shortKey(order.buyer_wallet)}</code></dd>
              <dt>Memo</dt><dd><code>{order.memo}</code></dd>
              {order.on_chain_tx_hash && (
                <>
                  <dt>Tx hash</dt>
                  <dd>
                    <a href={`https://stellar.expert/explorer/testnet/tx/${order.on_chain_tx_hash}`} target="_blank" rel="noreferrer">
                      <code>{shortKey(order.on_chain_tx_hash)}</code>
                    </a>
                  </dd>
                </>
              )}
            </dl>
            {order.status === 'awaiting_payment' && (
              <p className="muted">Waiting for payment to be detected on-chain (auto-refreshing).</p>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function StatusBadge({ status, reason }: { status: OrderStatus; reason: RejectReason }) {
  const m = STATUS_META[status]
  return (
    <p>
      <span className={`badge ${m.cls}`}>{m.label}</span>
      {status === 'rejected' && reason && (
        <span className="reject"> — {REJECT_LABEL[reason]}</span>
      )}
    </p>
  )
}

// ---- Order history -------------------------------------------------------

function History({ wallet, onTrack }: { wallet: string; onTrack: (id: string) => void }) {
  const [orders, setOrders] = useState<OrderDto[] | null>(null)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    if (!isStellarPubKey(wallet)) {
      setErr('Connected wallet is not a valid Stellar key')
      return
    }
    listOrders(wallet)
      .then((o) => { setOrders(o); setErr(null) })
      .catch((e) => setErr(e instanceof ApiError ? `${e.code}: ${e.message}` : 'Failed to load orders'))
  }, [wallet])

  if (err) return <div className="card"><p className="error">{err}</p></div>
  if (!orders) return <div className="card"><p className="muted">Loading…</p></div>
  if (orders.length === 0) return <div className="card"><p className="muted">No orders yet.</p></div>

  return (
    <div className="card">
      <h2>My orders</h2>
      <table className="orders">
        <thead>
          <tr><th>Product</th><th>Amount</th><th>Status</th><th></th></tr>
        </thead>
        <tbody>
          {orders.map((o) => (
            <tr key={o.id}>
              <td>{o.product_ref}</td>
              <td>{o.expected_amount} {o.asset_code}</td>
              <td><span className={`badge ${STATUS_META[o.status].cls}`}>{STATUS_META[o.status].label}</span></td>
              <td><button className="ghost" onClick={() => onTrack(o.id)}>View</button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

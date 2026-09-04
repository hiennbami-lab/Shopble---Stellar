import { useCallback, useEffect, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import {
  ApiError,
  createOrder,
  getOrder,
  listOrders,
  payUri,
  TESTNET_PASSPHRASE,
  type OrderDto,
  type PaymentInstruction,
} from './api'
import { connectWallet, disconnectWallet } from './wallet'
import { validateAmount, validateProductRef } from './validation'
import {
  CopyButton,
  LedgerRow,
  REJECT,
  Row,
  StatusPill,
  StatusTrack,
  shortKey,
} from './ui'

type View = { name: 'new' } | { name: 'orders' } | { name: 'order'; id: string }

const errText = (e: unknown, fallback: string) =>
  e instanceof ApiError ? e.message : e instanceof Error ? e.message : fallback

export default function App() {
  const [wallet, setWallet] = useState<string | null>(null)
  const [view, setView] = useState<View>({ name: 'new' })
  const [connecting, setConnecting] = useState(false)
  const [walletErr, setWalletErr] = useState<string | null>(null)
  // Instruction is only returned on create; kept so the slip renders immediately.
  const [fresh, setFresh] = useState<Record<string, PaymentInstruction>>({})

  const connect = async () => {
    setConnecting(true)
    setWalletErr(null)
    try {
      setWallet(await connectWallet())
      setView({ name: 'new' })
    } catch (e) {
      setWalletErr(errText(e, 'Could not connect to the wallet.'))
    } finally {
      setConnecting(false)
    }
  }

  const disconnect = () => {
    disconnectWallet().catch(() => {})
    setWallet(null)
    setFresh({})
  }

  return (
    <>
      <header className="topbar">
        <span className="brand">Shopble</span>
        <span className="chip-net">testnet</span>
        <span className="grow" />
        {wallet ? (
          <div className="chip-wallet">
            <span className="chip-dot" />
            <span title={wallet}>{shortKey(wallet, 4, 4)}</span>
            <button className="copy" onClick={disconnect}>Disconnect</button>
          </div>
        ) : null}
      </header>

      {!wallet ? (
        <section className="hero">
          <div>
          <h1>
            <span>Every order gets a memo.</span>
            <span>Every payment carries it back.</span>
          </h1>
          <p>
            Connect a Stellar wallet to create an order and get the payment slip that
            settles it. The memo on the slip is the order — that is what makes the
            payment provable against it.
          </p>
          {walletErr && <p className="alert" style={{ marginTop: 22, maxWidth: '46ch' }}>{walletErr}</p>}
          <div className="hero-actions">
            <button className="btn" onClick={connect} disabled={connecting}>
              {connecting ? 'Connecting' : 'Connect wallet'}
            </button>
            <span className="muted">Testnet only. No real funds move.</span>
          </div>
          </div>
          <aside className="specimen" aria-hidden="true">
            <p className="amount-label">Amount to pay</p>
            <p className="amount">
              25.5000000<span className="amount-unit">USDC</span>
            </p>
            <div className="memo">
              <p className="memo-label">Memo (required)</p>
              <div className="memo-value">
                <span className="mono">2iLKk8sQpZ9vTnR4bXcW1yFdA3m</span>
              </div>
              <p className="memo-note">
                Send this as a MEMO_TEXT memo. It is what ties the payment to the order.
              </p>
            </div>
          </aside>
        </section>
      ) : (
        <div className="shell">
          <nav className="rail">
            <button
              className={`nav${view.name === 'new' ? ' is-active' : ''}`}
              onClick={() => setView({ name: 'new' })}
            >
              New order
            </button>
            <button
              className={`nav${view.name !== 'new' ? ' is-active' : ''}`}
              onClick={() => setView({ name: 'orders' })}
            >
              Orders
            </button>
          </nav>

          <main>
            {view.name === 'new' && (
              <NewOrder
                wallet={wallet}
                onCreated={(order, instruction) => {
                  setFresh((f) => ({ ...f, [order.id]: instruction }))
                  setView({ name: 'order', id: order.id })
                }}
              />
            )}
            {view.name === 'orders' && (
              <Orders
                wallet={wallet}
                onOpen={(id) => setView({ name: 'order', id })}
                onNew={() => setView({ name: 'new' })}
              />
            )}
            {view.name === 'order' && (
              <Order
                id={view.id}
                instruction={fresh[view.id]}
                onBack={() => setView({ name: 'orders' })}
              />
            )}
          </main>
        </div>
      )}
    </>
  )
}

// ---------------------------------------------------------------- new order

function NewOrder({
  wallet,
  onCreated,
}: {
  wallet: string
  onCreated: (order: OrderDto, instruction: PaymentInstruction) => void
}) {
  const [productRef, setProductRef] = useState('')
  const [amount, setAmount] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    const problem = validateProductRef(productRef) ?? validateAmount(amount)
    if (problem) {
      setErr(problem)
      return
    }
    setErr(null)
    setBusy(true)
    try {
      const res = await createOrder({
        product_ref: productRef.trim(),
        expected_amount: amount.trim(),
        buyer_wallet: wallet,
      })
      onCreated(res.order, res.instruction)
    } catch (e) {
      setErr(errText(e, 'Could not create the order.'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form className="panel" onSubmit={submit}>
      <div className="panel-head">
        <h2 className="panel-title">New order</h2>
        <span className="muted" style={{ fontSize: 13 }}>
          Paying from <span className="mono" title={wallet}>{shortKey(wallet, 4, 4)}</span>
        </span>
      </div>

      {err && <p className="alert">{err}</p>}

      <label className="field">
        <span className="field-label">Product reference</span>
        <input
          className="field-input"
          value={productRef}
          onChange={(e) => setProductRef(e.target.value)}
          placeholder="sku-1024"
        />
        <span className="field-hint">Whatever identifies the item on your side.</span>
      </label>

      <label className="field">
        <span className="field-label">Amount</span>
        <span className="field-affix">
          <input
            className="field-input"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="25.5"
            inputMode="decimal"
          />
          <span className="field-unit">USDC</span>
        </span>
        <span className="field-hint">Up to 7 decimal places.</span>
      </label>

      <button className="btn" type="submit" disabled={busy}>
        {busy ? 'Creating order' : 'Create order'}
      </button>
    </form>
  )
}

// ---------------------------------------------------------------- one order

function Order({
  id,
  instruction,
  onBack,
}: {
  id: string
  instruction?: PaymentInstruction
  onBack: () => void
}) {
  const [order, setOrder] = useState<OrderDto | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      setOrder(await getOrder(id))
      setErr(null)
    } catch (e) {
      setErr(errText(e, 'Could not load this order.'))
    }
  }, [id])

  useEffect(() => {
    load()
    // Poll while the order can still change. Stops once it settles either way.
    const t = setInterval(() => {
      setOrder((o) => {
        if (o && (o.status === 'validated' || o.status === 'rejected')) return o
        load()
        return o
      })
    }, 4000)
    return () => clearInterval(t)
  }, [load])

  const open = () => (
    <>
      <button className="back" onClick={onBack}>Back to orders</button>
      {err && <p className="alert">{err}</p>}
    </>
  )

  if (!order) {
    return (
      <>
        {open()}
        <div className="panel"><p className="muted">Loading order.</p></div>
      </>
    )
  }

  const payable = order.status === 'awaiting_payment'
  const uri = instruction?.uri ?? payUri(order)

  return (
    <>
      {open()}

      <section className="panel">
        <div className="panel-head">
          <h2 className="panel-title">{order.product_ref}</h2>
          {!payable && <StatusPill status={order.status} />}
        </div>

        <StatusTrack status={order.status} />

        {order.status === 'rejected' && order.reject_reason && (
          <p className="reject-note">{REJECT[order.reject_reason]}</p>
        )}

        {payable && (
          <>
          <div className="slip">
            <div>
              <p className="amount-label">Amount to pay</p>
              <p className="amount">
                {order.expected_amount}
                <span className="amount-unit">{order.asset_code}</span>
              </p>

              <div className="memo">
                <p className="memo-label">Memo (required)</p>
                <div className="memo-value">
                  <span className="mono">{order.memo}</span>
                  <CopyButton value={order.memo} label="memo" />
                </div>
                <p className="memo-note">
                  Send this as a MEMO_TEXT memo. Without it the payment cannot be
                  matched to this order, even if the amount is right.
                </p>
              </div>
            </div>

            <div className="qr">
              <QRCodeSVG value={uri} size={166} level="M" bgColor="#ffffff" fgColor="#101828" />
              <p className="qr-cap">Scan to pay</p>
            </div>
          </div>

          <div className="actions">
            <a className="btn" href={uri}>Open in wallet</a>
            <CopyButton value={uri} label="payment link" text="Copy payment link" />
            <span className="live"><span className="live-dot" />Checking the ledger</span>
          </div>
          </>
        )}
      </section>

      <section className="section">
        <h3 className="section-title">Details</h3>
        <div className="ledger">
          <LedgerRow label="Order" value={order.id} />
          <Row label="Amount">
            <span className="mono">{order.expected_amount} {order.asset_code}</span>
          </Row>
          <LedgerRow label="Destination" value={order.destination_account} short />
          <LedgerRow label="Asset issuer" value={order.asset_issuer} short />
          <LedgerRow label="Buyer wallet" value={order.buyer_wallet} short />
          <Row label="Network">
            <span className="mono">{instruction?.network ?? TESTNET_PASSPHRASE}</span>
          </Row>
          <Row label="Created">
            <span>{new Date(order.created_at * 1000).toLocaleString()}</span>
          </Row>
          {order.on_chain_tx_hash && (
            <Row label="Transaction">
              <a
                className="mono"
                href={`https://stellar.expert/explorer/testnet/tx/${order.on_chain_tx_hash}`}
                target="_blank"
                rel="noreferrer"
              >
                {shortKey(order.on_chain_tx_hash, 10, 10)}
              </a>
              <CopyButton value={order.on_chain_tx_hash} label="transaction hash" />
            </Row>
          )}
        </div>
      </section>
    </>
  )
}

// ----------------------------------------------------------------- history

function Orders({
  wallet,
  onOpen,
  onNew,
}: {
  wallet: string
  onOpen: (id: string) => void
  onNew: () => void
}) {
  const [orders, setOrders] = useState<OrderDto[] | null>(null)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    listOrders(wallet)
      .then((o) => { setOrders(o); setErr(null) })
      .catch((e) => setErr(errText(e, 'Could not load your orders.')))
  }, [wallet])

  return (
    <section className="panel">
      <div className="panel-head"><h2 className="panel-title">Orders</h2></div>

      {err && <p className="alert">{err}</p>}
      {!err && !orders && <p className="muted">Loading orders.</p>}

      {orders && orders.length === 0 && (
        <div className="empty">
          <h3>No orders yet</h3>
          <p>Create an order to get its payment slip.</p>
          <button className="btn" onClick={onNew}>New order</button>
        </div>
      )}

      {orders && orders.length > 0 && (
        <div className="table-scroll">
          <table className="table">
            <thead>
              <tr>
                <th>Product</th>
                <th>Amount</th>
                <th>Created</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.id} onClick={() => onOpen(o.id)}>
                  <td>{o.product_ref}</td>
                  <td><span className="mono">{o.expected_amount} {o.asset_code}</span></td>
                  <td className="muted">{new Date(o.created_at * 1000).toLocaleDateString()}</td>
                  <td><StatusPill status={o.status} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

import { useState, type ReactNode } from 'react'
import type { OrderStatus, RejectReason, Verdict } from './api'

export const shortKey = (k: string, head = 6, tail = 6) =>
  k.length > head + tail + 1 ? `${k.slice(0, head)}…${k.slice(-tail)}` : k

export const STATUS: Record<OrderStatus, { label: string; tone: string }> = {
  awaiting_payment: { label: 'Awaiting payment', tone: 'wait' },
  payment_detected: { label: 'Payment detected', tone: 'det' },
  validated: { label: 'Validated', tone: 'ok' },
  rejected: { label: 'Rejected', tone: 'bad' },
}

// Written for the buyer, not the matcher: what went wrong, in their terms.
export const REJECT: Record<Exclude<RejectReason, ''>, string> = {
  underpayment: 'The amount paid was less than the order.',
  wrong_asset: 'The payment used a different asset.',
  wrong_destination: 'The payment went to a different account.',
  wrong_buyer_wallet: 'The payment came from a different wallet.',
  invalid_memo: 'The memo did not match this order.',
}

export const VERDICT: Record<Verdict, { label: string; tone: string }> = {
  matched: { label: 'Matched', tone: 'ok' },
  rejected: { label: 'Rejected', tone: 'bad' },
  duplicate: { label: 'Duplicate', tone: 'det' },
}

// The six fields the matcher compares, in the order the backend returns them.
export const FIELD_LABEL: Record<string, string> = {
  amount: 'Amount',
  asset_code: 'Asset',
  asset_issuer: 'Asset issuer',
  destination_account: 'Destination',
  buyer_wallet: 'Buyer wallet',
  memo: 'Memo',
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`
  if (seconds < 86400) return `${Math.round(seconds / 3600)}h`
  return `${Math.round(seconds / 86400)}d`
}

export function StatusPill({ status }: { status: OrderStatus }) {
  const s = STATUS[status]
  return <span className={`pill pill-${s.tone}`}>{s.label}</span>
}

export function CopyButton({ value, label, text }: { value: string; label: string; text?: string }) {
  const [done, setDone] = useState(false)
  return (
    <button
      type="button"
      className={text ? 'copy copy-wide' : 'copy'}
      aria-label={`Copy ${label}`}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value)
          setDone(true)
          setTimeout(() => setDone(false), 1400)
        } catch {
          // Clipboard blocked (insecure origin / permission) — the value stays selectable.
        }
      }}
    >
      {done ? 'Copied' : (text ?? 'Copy')}
    </button>
  )
}

export function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="lrow">
      <div className="lrow-label">{label}</div>
      <div className="lrow-value">{children}</div>
    </div>
  )
}

export function LedgerRow({ label, value, short }: { label: string; value: string; short?: boolean }) {
  return (
    <Row label={label}>
      <span className="mono" title={value}>{short ? shortKey(value) : value}</span>
      <CopyButton value={value} label={label} />
    </Row>
  )
}

const STEPS: OrderStatus[] = ['awaiting_payment', 'payment_detected', 'validated']

// The state machine, drawn. A rejected order fills the last node red instead of green.
export function StatusTrack({ status }: { status: OrderStatus }) {
  const rejected = status === 'rejected'
  const current = rejected ? 1 : STEPS.indexOf(status)
  return (
    <ol className="track">
      {STEPS.map((step, i) => {
        const last = i === STEPS.length - 1
        const isRejectSlot = rejected && last
        // A rejected order has settled — nothing on it is still in progress.
        const state = isRejectSlot
          ? 'bad'
          : i < current || rejected
            ? 'done'
            : i === current
              ? 'current'
              : 'todo'
        return (
          <li key={step} className={`track-step is-${state}`}>
            <span className="track-dot" />
            <span>{isRejectSlot ? 'Rejected' : STATUS[step].label}</span>
            {!last && <span className="track-line" />}
          </li>
        )
      })}
    </ol>
  )
}

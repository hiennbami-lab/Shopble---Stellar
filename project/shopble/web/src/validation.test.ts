import { describe, it, expect } from 'vitest'
import { isStellarPubKey, validateAmount, validateProductRef } from './validation'

const G = 'G' + 'A'.repeat(55) // 56 chars

describe('isStellarPubKey', () => {
  it('accepts a G + 55 base32 key', () => expect(isStellarPubKey(G)).toBe(true))
  it('rejects wrong length', () => expect(isStellarPubKey('G' + 'A'.repeat(54))).toBe(false))
  it('rejects wrong prefix', () => expect(isStellarPubKey('M' + 'A'.repeat(55))).toBe(false))
  it('rejects non-base32 chars (0,1,8,9)', () => expect(isStellarPubKey('G' + '0'.repeat(55))).toBe(false))
})

describe('validateAmount', () => {
  it('accepts a plain decimal', () => expect(validateAmount('25.5')).toBeNull())
  it('accepts 7 dp', () => expect(validateAmount('0.0000001')).toBeNull())
  it('rejects 8 dp', () => expect(validateAmount('0.00000001')).not.toBeNull())
  it('rejects zero', () => expect(validateAmount('0')).not.toBeNull())
  it('rejects negatives', () => expect(validateAmount('-1')).not.toBeNull())
  it('rejects non-numbers', () => expect(validateAmount('abc')).not.toBeNull())
  it('rejects empty', () => expect(validateAmount('  ')).not.toBeNull())
  it('rejects over the int64 ceiling', () => expect(validateAmount('922337203686')).not.toBeNull())
})

describe('validateProductRef', () => {
  it('accepts a normal ref', () => expect(validateProductRef('sku-1024')).toBeNull())
  it('rejects empty', () => expect(validateProductRef('')).not.toBeNull())
  it('rejects > 255 chars', () => expect(validateProductRef('x'.repeat(256))).not.toBeNull())
})

import { formatDuration } from './ui'

describe('formatDuration', () => {
  it('keeps seconds under a minute', () => expect(formatDuration(4)).toBe('4s'))
  it('rounds to minutes', () => expect(formatDuration(150)).toBe('3m'))
  it('rounds to hours', () => expect(formatDuration(7200)).toBe('2h'))
  it('rounds to days', () => expect(formatDuration(3924875)).toBe('45d'))
})

import { payErrorText } from './pay'

const horizonError = (result_codes: unknown) => ({ response: { data: { extras: { result_codes } } } })

describe('payErrorText', () => {
  it('explains a known operation code and keeps the raw code', () => {
    const t = payErrorText(horizonError({ transaction: 'tx_failed', operations: ['op_no_trust'] }))
    expect(t).toContain('no trustline')
    expect(t).toContain('op_no_trust')
  })
  it('skips op_success to find the real failure', () =>
    expect(payErrorText(horizonError({ operations: ['op_success', 'op_underfunded'] }))).toContain('op_underfunded'))
  it('falls back to the transaction code when no op code explains it', () =>
    expect(payErrorText(horizonError({ transaction: 'tx_bad_auth' }))).toBe('tx_bad_auth'))
  it('falls back to the error message when Horizon gave no codes', () =>
    expect(payErrorText(new Error('network down'))).toBe('network down'))
})

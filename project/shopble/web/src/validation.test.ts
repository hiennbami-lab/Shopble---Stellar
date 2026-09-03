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

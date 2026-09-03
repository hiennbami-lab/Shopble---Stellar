// Stellar Wallets Kit (v2, static API) — testnet only. authModal() shows the
// wallet picker (Freighter + all zero-config SEP-43 modules), sets the selected
// wallet, and returns the account public key — the buyer identity.
import { StellarWalletsKit, Networks } from '@creit.tech/stellar-wallets-kit'
import { defaultModules } from '@creit.tech/stellar-wallets-kit/modules/utils'
import { FREIGHTER_ID } from '@creit.tech/stellar-wallets-kit/modules/freighter'

StellarWalletsKit.init({
  network: Networks.TESTNET,
  selectedWalletId: FREIGHTER_ID,
  modules: defaultModules(),
})

export async function connectWallet(): Promise<string> {
  const { address } = await StellarWalletsKit.authModal()
  return address
}

export function disconnectWallet(): Promise<void> {
  return StellarWalletsKit.disconnect()
}

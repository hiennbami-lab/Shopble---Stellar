package gmeta

import "github.com/shopspring/decimal"

type BlockchainNetwork string

type Currency string

type BlockchainType string

type BlockchainTxnStatus int8

type CurrencyAmount struct {
	Currency Currency        `json:"currency"`
	Value    decimal.Decimal `json:"value"`
}

type Coin interface {
	GetCurrency() Currency
	GetNetwork() BlockchainNetwork
}

type BlockchainCoinIndex struct {
	Currency Currency          `json:"currency"`
	Network  BlockchainNetwork `json:"network"`
}

func (bi *BlockchainCoinIndex) GetCurrency() Currency {
	return bi.Currency
}

func (bi *BlockchainCoinIndex) GetNetwork() BlockchainNetwork {
	return bi.Network
}

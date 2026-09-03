package models

import "github.com/shopspring/decimal"

type OrderStatus string

// State machine của SOW: awaiting_payment → payment_detected → validated | rejected.
// Chỉ `validated` mới cho phép fulfilment.
const (
	OrderStatusAwaitingPayment OrderStatus = "awaiting_payment"
	OrderStatusPaymentDetected OrderStatus = "payment_detected"
	OrderStatusValidated       OrderStatus = "validated"
	OrderStatusRejected        OrderStatus = "rejected"
)

// RejectReason — 5 lý do từ chối có tên (SOW Deliverable 2). Enum đóng: matcher phải
// trả về đúng một trong số này, không được có nhánh "rejected" nào không tên, vì
// chính cái tên mới làm kết quả kiểm chứng được thay vì chỉ nghe hợp lý.
type RejectReason string

const (
	RejectUnderpayment     RejectReason = "underpayment"
	RejectWrongAsset       RejectReason = "wrong_asset"
	RejectWrongDestination RejectReason = "wrong_destination"
	RejectWrongBuyerWallet RejectReason = "wrong_buyer_wallet"
	RejectInvalidMemo      RejectReason = "invalid_memo"
)

func ValidRejectReason(r RejectReason) bool {
	switch r {
	case RejectUnderpayment, RejectWrongAsset, RejectWrongDestination,
		RejectWrongBuyerWallet, RejectInvalidMemo:
		return true
	}
	return false
}

// orderTransitions — cạnh hợp lệ duy nhất của state machine.
var orderTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusAwaitingPayment: {OrderStatusPaymentDetected},
	OrderStatusPaymentDetected: {OrderStatusValidated, OrderStatusRejected},
	OrderStatusValidated:       {},
	OrderStatusRejected:        {},
}

func CanTransition(from, to OrderStatus) bool {
	for _, allowed := range orderTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// OrderFinal — order đã chốt, không nhận verdict mới. Payment tới sau khi order chốt
// là duplicate và phải được đánh dấu riêng, không được ghi đè verdict cũ.
func OrderFinal(s OrderStatus) bool {
	return s == OrderStatusValidated || s == OrderStatusRejected
}

// OrderIntent — 7 field bắt buộc lưu theo SOW Deliverable 1:
// product context, expected amount, asset (code+issuer), destination, memo,
// buyer wallet, status.
type OrderIntent struct {
	Id string `gorm:"column:id;primaryKey;type:varchar(32)"`
	// ProductRef — product/source context. Free-form, do client truyền lên.
	ProductRef string `gorm:"column:product_ref;type:varchar(255);not null"`
	// Stellar tối đa 7 chữ số thập phân; numeric(23,7) phủ hết dải int64 stroops.
	ExpectedAmount     decimal.Decimal `gorm:"column:expected_amount;type:numeric(23,7);not null"`
	AssetCode          string          `gorm:"column:asset_code;type:varchar(12);not null"`
	AssetIssuer        string          `gorm:"column:asset_issuer;type:varchar(56);not null"`
	DestinationAccount string          `gorm:"column:destination_account;type:varchar(56);not null"`
	// Memo — reference nối payment với order. Unique, ≤28 byte (giới hạn MEMO_TEXT).
	Memo string `gorm:"column:memo;type:varchar(28);uniqueIndex;not null"`
	// BuyerWallet — địa chỉ ví đã connect. Ở scope này nó chính là danh tính của buyer,
	// không có tài khoản user riêng.
	BuyerWallet  string       `gorm:"column:buyer_wallet;type:varchar(56);index;not null"`
	Status       OrderStatus  `gorm:"column:status;type:varchar(32);default:'awaiting_payment';not null;index"`
	RejectReason RejectReason `gorm:"column:reject_reason;type:varchar(32)"`
	// OnChainTxHash — transaction ghi verdict lên Soroban contract (Deliverable 3).
	OnChainTxHash string `gorm:"column:on_chain_tx_hash;type:varchar(80)"`
	CreatedAt     int64  `gorm:"column:created_at;autoCreateTime;index"`
	UpdatedAt     int64  `gorm:"column:updated_at;autoUpdateTime"`
}

func (OrderIntent) TableName() string { return "shopble_order_intent" }

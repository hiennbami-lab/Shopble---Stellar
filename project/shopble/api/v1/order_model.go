package v1

import (
	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"
)

type CreateOrderRequest struct {
	// ProductRef — product/source context của order.
	ProductRef string `json:"product_ref" binding:"required,max=255" example:"sku-1024"`
	// ExpectedAmount — số tiền kỳ vọng, dạng chuỗi thập phân, tối đa 7 chữ số lẻ.
	ExpectedAmount string `json:"expected_amount" binding:"required" example:"25.5"`
	// BuyerWallet — địa chỉ ví Stellar đã connect (G..., 56 ký tự).
	BuyerWallet string `json:"buyer_wallet" binding:"required" example:"GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"`
}

// OrderDto — 7 field lưu trữ theo SOW, cộng verdict nếu đã có.
type OrderDto struct {
	Id                 string `json:"id"`
	ProductRef         string `json:"product_ref"`
	ExpectedAmount     string `json:"expected_amount" example:"25.5000000"`
	AssetCode          string `json:"asset_code"`
	AssetIssuer        string `json:"asset_issuer"`
	DestinationAccount string `json:"destination_account"`
	Memo               string `json:"memo"`
	BuyerWallet        string `json:"buyer_wallet"`
	Status             string `json:"status" example:"awaiting_payment"`
	RejectReason       string `json:"reject_reason,omitempty"`
	OnChainTxHash      string `json:"on_chain_tx_hash,omitempty"`
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`
}

type CreateOrderData struct {
	Order       OrderDto                       `json:"order"`
	Instruction *libstellar.PaymentInstruction `json:"instruction"`
}

type CreateOrderEnvelope struct {
	Status string          `json:"status" example:"ok"`
	Data   CreateOrderData `json:"data"`
}

type OrderEnvelope struct {
	Status string   `json:"status" example:"ok"`
	Data   OrderDto `json:"data"`
}

type OrderListEnvelope struct {
	Status string     `json:"status" example:"ok"`
	Data   []OrderDto `json:"data"`
}

type ErrorData struct {
	ErrorCode string `json:"error_code" example:"DATA_INVALID"`
	Message   string `json:"message"`
}

type ErrorResponse struct {
	Status string    `json:"status" example:"error"`
	Data   ErrorData `json:"data"`
}

func toOrderDto(row *models.OrderIntent) OrderDto {
	return OrderDto{
		Id:                 row.Id,
		ProductRef:         row.ProductRef,
		ExpectedAmount:     row.ExpectedAmount.StringFixed(libstellar.StellarDecimals),
		AssetCode:          row.AssetCode,
		AssetIssuer:        row.AssetIssuer,
		DestinationAccount: row.DestinationAccount,
		Memo:               row.Memo,
		BuyerWallet:        row.BuyerWallet,
		Status:             string(row.Status),
		RejectReason:       string(row.RejectReason),
		OnChainTxHash:      row.OnChainTxHash,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

// EvidenceDto — bằng chứng thô một payment, cộng link explorer và detection latency
// (hai thứ SOW Deliverable 2 bắt có trong bảng kết quả).
type EvidenceDto struct {
	Id                 string `json:"id"`
	OpId               string `json:"op_id"`
	TxHash             string `json:"tx_hash"`
	ExplorerUrl        string `json:"explorer_url"`
	SourceAccount      string `json:"source_account"`
	DestinationAccount string `json:"destination_account"`
	AssetCode          string `json:"asset_code"`
	AssetIssuer        string `json:"asset_issuer"`
	Amount             string `json:"amount"`
	Memo               string `json:"memo"`
	LedgerCloseAt      int64  `json:"ledger_close_at"`
	CapturedAt         int64  `json:"captured_at"`
	// DetectionLatencySeconds — captured_at trừ ledger_close_at.
	DetectionLatencySeconds int64  `json:"detection_latency_seconds"`
	OrderId                 string `json:"order_id,omitempty"`
	Verdict                 string `json:"verdict" example:"matched"`
	RejectReason            string `json:"reject_reason,omitempty"`
}

// FieldComparison — một field expected đặt cạnh observed.
type FieldComparison struct {
	Field    string `json:"field" example:"asset_code"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	Match    bool   `json:"match"`
}

type OrderEvidenceDto struct {
	Evidence   EvidenceDto       `json:"evidence"`
	Comparison []FieldComparison `json:"comparison"`
}

type OrderEvidenceData struct {
	Order    OrderDto           `json:"order"`
	Evidence []OrderEvidenceDto `json:"evidence"`
}

type EvidenceListEnvelope struct {
	Status string        `json:"status" example:"ok"`
	Data   []EvidenceDto `json:"data"`
}

type OrderEvidenceEnvelope struct {
	Status string            `json:"status" example:"ok"`
	Data   OrderEvidenceData `json:"data"`
}

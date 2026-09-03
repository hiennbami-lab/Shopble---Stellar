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

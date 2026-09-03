package watcher

import (
	"shopble/project/shopble/models"

	"github.com/shopspring/decimal"
)

// Observed — giá trị thô của một payment đã đọc từ ledger, dạng matcher cần. Tách khỏi
// HorizonPayment để matcher là hàm thuần, không phụ thuộc Horizon và test được thẳng.
type Observed struct {
	Source      string
	Destination string
	AssetCode   string
	AssetIssuer string
	Amount      decimal.Decimal
	Memo        string
}

// Match — engine đối chiếu MỘT payment với order tra theo memo. Thuần: không DB, không IO.
//
//   - order == nil: memo không khớp order nào → rejected(invalid_memo). Không có order để đổi trạng thái.
//   - order đã chốt (validated/rejected): payment tới sau → duplicate, KHÔNG ghi đè verdict cũ.
//
// Thứ tự kiểm tra quyết định reason nào thắng khi sai nhiều thứ:
// destination → asset → buyer wallet → amount. Kiểm tra đầu tiên fail là reason có tên.
func Match(o Observed, order *models.OrderIntent) (models.Verdict, models.RejectReason) {
	if order == nil {
		return models.VerdictRejected, models.RejectInvalidMemo
	}
	if models.OrderFinal(order.Status) {
		return models.VerdictDuplicate, ""
	}
	if o.Destination != order.DestinationAccount {
		return models.VerdictRejected, models.RejectWrongDestination
	}
	if o.AssetCode != order.AssetCode || o.AssetIssuer != order.AssetIssuer {
		return models.VerdictRejected, models.RejectWrongAsset
	}
	if o.Source != order.BuyerWallet {
		return models.VerdictRejected, models.RejectWrongBuyerWallet
	}
	// Underpayment nếu THIẾU. Trả dư (overpayment) vẫn matched — SOW chỉ đặt tên cho thiếu.
	if o.Amount.LessThan(order.ExpectedAmount) {
		return models.VerdictRejected, models.RejectUnderpayment
	}
	return models.VerdictMatched, ""
}

// Terminal — trạng thái order chốt tương ứng với verdict của payment.
// Chỉ dùng cho matched/rejected; duplicate không đổi trạng thái order.
func Terminal(v models.Verdict) models.OrderStatus {
	if v == models.VerdictMatched {
		return models.OrderStatusValidated
	}
	return models.OrderStatusRejected
}

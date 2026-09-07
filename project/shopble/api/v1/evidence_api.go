package v1

import (
	"net/http"
	"strconv"

	"shopble/database"
	"shopble/project/shopble/api/apierr"
	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"

	"github.com/gin-gonic/gin"
)

// explorerBase — Stellar Expert testnet. SOW yêu cầu mỗi transaction trong bảng kết quả
// phải có link explorer bấm được, nên backend trả luôn link thay vì bắt người đọc tự ghép.
const explorerBase = "https://stellar.expert/explorer/testnet/tx/"

func toEvidenceDto(row *models.PaymentEvidence) EvidenceDto {
	// Detection latency = lúc backend nhìn thấy payment trừ lúc ledger đóng.
	// Đây là con số SOW bắt báo cáo cho từng transaction.
	latency := row.CapturedAt - row.LedgerCloseAt
	if latency < 0 {
		latency = 0
	}
	return EvidenceDto{
		Id:                      row.Id,
		OpId:                    row.OpId,
		TxHash:                  row.TxHash,
		ExplorerUrl:             explorerBase + row.TxHash,
		SourceAccount:           row.SourceAccount,
		DestinationAccount:      row.DestinationAccount,
		AssetCode:               row.AssetCode,
		AssetIssuer:             row.AssetIssuer,
		Amount:                  row.Amount.StringFixed(libstellar.StellarDecimals),
		Memo:                    row.Memo,
		LedgerCloseAt:           row.LedgerCloseAt,
		CapturedAt:              row.CapturedAt,
		DetectionLatencySeconds: latency,
		OrderId:                 row.OrderId,
		Verdict:                 string(row.Verdict),
		RejectReason:            string(row.RejectReason),
	}
}

// ListEvidence godoc
// @Summary      Liệt kê payment evidence thô đọc từ ledger
// @Description  Nguồn cho bảng kết quả 15 transaction của SOW Deliverable 2: mỗi dòng có tx hash, link Stellar Expert, verdict và detection latency.
// @Tags         evidence
// @Produce      json
// @Param        order_id  query     string  false  "Lọc theo order"
// @Param        verdict   query     string  false  "Lọc theo verdict: matched | rejected | duplicate"
// @Param        limit     query     int     false  "Mặc định 50, tối đa 200"
// @Success      200  {object}  EvidenceListEnvelope
// @Failure      400  {object}  ErrorResponse
// @Router       /evidence [get]
func ListEvidence(c *gin.Context) {
	limit := 50
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = min(v, 200)
	}

	q := database.GetDb().WithContext(c).Model(&models.PaymentEvidence{})
	if orderId := c.Query("order_id"); orderId != "" {
		q = q.Where("order_id = ?", orderId)
	}
	if verdict := c.Query("verdict"); verdict != "" {
		q = q.Where("verdict = ?", verdict)
	}

	var rows []models.PaymentEvidence
	if err := q.Order("captured_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		apierr.Internal(c, "lookup failed: "+err.Error())
		return
	}

	items := make([]EvidenceDto, 0, len(rows))
	for i := range rows {
		items = append(items, toEvidenceDto(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "data": items})
}

// compare — expected (theo order) đặt cạnh observed (theo ledger), từng field một.
//
// Đây là thứ làm verdict kiểm chứng được thay vì phải tin backend: người review nhìn
// đúng hai cột và tự thấy field nào lệch, không cần đọc code matcher.
func compare(order *models.OrderIntent, ev *models.PaymentEvidence) []FieldComparison {
	pairs := []struct {
		field    string
		expected string
		observed string
	}{
		{"amount", order.ExpectedAmount.StringFixed(libstellar.StellarDecimals), ev.Amount.StringFixed(libstellar.StellarDecimals)},
		{"asset_code", order.AssetCode, ev.AssetCode},
		{"asset_issuer", order.AssetIssuer, ev.AssetIssuer},
		{"destination_account", order.DestinationAccount, ev.DestinationAccount},
		{"buyer_wallet", order.BuyerWallet, ev.SourceAccount},
		{"memo", order.Memo, ev.Memo},
	}
	out := make([]FieldComparison, 0, len(pairs))
	for _, p := range pairs {
		match := p.expected == p.observed
		// Trả dư vẫn hợp lệ: chỉ THIẾU mới là underpayment.
		if p.field == "amount" && !match && ev.Amount.GreaterThan(order.ExpectedAmount) {
			match = true
		}
		out = append(out, FieldComparison{
			Field: p.field, Expected: p.expected, Observed: p.observed, Match: match,
		})
	}
	return out
}

// GetOrderEvidence godoc
// @Summary      Order kèm bằng chứng, dạng expected-versus-observed
// @Description  View cho người review theo SOW: order intent, mọi payment đã quan sát cho order đó, và so sánh từng field.
// @Tags         evidence
// @Produce      json
// @Param        id   path      string  true  "Order id"
// @Success      200  {object}  OrderEvidenceEnvelope
// @Failure      404  {object}  ErrorResponse
// @Router       /orders/{id}/evidence [get]
func GetOrderEvidence(c *gin.Context) {
	var order models.OrderIntent
	if err := database.GetDb().WithContext(c).Where("id = ?", c.Param("id")).Take(&order).Error; err != nil {
		apierr.NotFound(c, "order not found")
		return
	}

	var rows []models.PaymentEvidence
	if err := database.GetDb().WithContext(c).
		Where("order_id = ?", order.Id).
		Order("captured_at ASC").Find(&rows).Error; err != nil {
		apierr.Internal(c, "lookup failed: "+err.Error())
		return
	}

	items := make([]OrderEvidenceDto, 0, len(rows))
	for i := range rows {
		items = append(items, OrderEvidenceDto{
			Evidence:   toEvidenceDto(&rows[i]),
			Comparison: compare(&order, &rows[i]),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "data": OrderEvidenceData{
		Order:    toOrderDto(&order),
		Evidence: items,
	}})
}

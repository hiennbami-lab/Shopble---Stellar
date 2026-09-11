package libstellar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Horizon có REST + JSON thuần, nên watcher poll thẳng bằng net/http thay vì kéo cả
// module stellar/go vào go.mod (init commit đã cố tình bỏ go-ethereum để giữ deps gọn).
//
// Endpoint: GET /accounts/{dest}/payments?cursor=&order=asc&limit=&join=transactions
// `join=transactions` nhúng luôn transaction vào mỗi record để lấy memo — memo nằm ở
// transaction, KHÔNG nằm ở operation.
//
// Nhận cả `payment` lẫn `path_payment_strict_send|receive`: buyer route qua DEX vẫn là
// buyer trả tiền, bỏ qua thì order treo ở awaiting_payment vĩnh viễn mà không báo gì.

// HorizonPayment — một payment operation quan sát từ ledger, đã phẳng hoá field cần dùng.
type HorizonPayment struct {
	Id              string
	PagingToken     string
	Type            string
	TransactionHash string
	From            string
	To              string
	AssetCode       string // "XLM" nếu native
	AssetIssuer     string // rỗng nếu native
	Amount          string
	Memo            string // chỉ có nghĩa khi memo_type == "text"; rỗng nếu khác
	LedgerCloseAt   int64  // unix seconds, từ created_at
}

// horizonPage — chỉ những field JSON ta thực sự đọc.
type horizonPage struct {
	Embedded struct {
		Records []struct {
			Id              string `json:"id"`
			PagingToken     string `json:"paging_token"`
			Type            string `json:"type"`
			TransactionHash string `json:"transaction_hash"`
			From            string `json:"from"`
			To              string `json:"to"`
			AssetType       string `json:"asset_type"`
			AssetCode       string `json:"asset_code"`
			AssetIssuer     string `json:"asset_issuer"`
			Amount          string `json:"amount"`
			CreatedAt       string `json:"created_at"`
			Transaction     struct {
				Memo     string `json:"memo"`
				MemoType string `json:"memo_type"`
			} `json:"transaction"`
		} `json:"records"`
	} `json:"_embedded"`
}

var horizonHttp = &http.Client{Timeout: 30 * time.Second}

// FetchPayments — đọc một trang payment operations của destination account từ cursor.
// order=asc để xử lý theo thứ tự ledger; cursor rỗng = đọc từ đầu lịch sử account.
func (c *StellarConfig) FetchPayments(ctx context.Context, cursor string, limit int) ([]HorizonPayment, error) {
	q := url.Values{}
	q.Set("order", "asc")
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("join", "transactions")
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	endpoint := fmt.Sprintf("%s/accounts/%s/payments?%s",
		strings.TrimRight(c.HorizonUrl, "/"), c.DestinationAccount, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := horizonHttp.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("horizon payments: status %d", resp.StatusCode)
	}

	var page horizonPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("horizon payments: decode: %w", err)
	}

	out := make([]HorizonPayment, 0, len(page.Embedded.Records))
	for _, r := range page.Embedded.Records {
		// Endpoint /payments trả cả create_account, account_merge — không phải payment
		// nào cũng là tiền hàng. Với path payment thì `asset_*`/`amount` đã là thứ
		// destination NHẬN được (source_asset là thứ buyer bỏ ra, không liên quan),
		// nên matcher đọc y hệt classic payment.
		if r.Type != "payment" && !strings.HasPrefix(r.Type, "path_payment") {
			continue
		}
		code := r.AssetCode
		if r.AssetType == "native" {
			code = "XLM"
		}
		memo := ""
		if r.Transaction.MemoType == "text" {
			memo = r.Transaction.Memo
		}
		var closeAt int64
		if t, err := time.Parse(time.RFC3339, r.CreatedAt); err == nil {
			closeAt = t.Unix()
		}
		out = append(out, HorizonPayment{
			Id:              r.Id,
			PagingToken:     r.PagingToken,
			Type:            r.Type,
			TransactionHash: r.TransactionHash,
			From:            r.From,
			To:              r.To,
			AssetCode:       code,
			AssetIssuer:     r.AssetIssuer,
			Amount:          r.Amount,
			Memo:            memo,
			LedgerCloseAt:   closeAt,
		})
	}
	return out, nil
}

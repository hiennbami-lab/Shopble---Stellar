package libstellar

import (
	"fmt"
	"net/url"

	"github.com/shopspring/decimal"
)

// StellarDecimals — Stellar lưu số tiền là int64 stroops, 1 unit = 10^7 stroops.
// Mọi số vượt quá 7 chữ số thập phân sẽ bị làm tròn ở đâu đó trên đường đi, và
// một amount làm tròn khác amount kỳ vọng là một underpayment giả.
const StellarDecimals = 7

// MaxAmount — trần int64 stroops quy về unit.
var MaxAmount = decimal.RequireFromString("922337203685.4775807")

// PaymentInstruction — lệnh thanh toán sinh ra cho một order intent. Đây là hợp đồng
// giữa backend và ví của buyer: trả sai bất kỳ field nào cũng dẫn tới một rejection
// có tên trong matcher.
type PaymentInstruction struct {
	Destination string `json:"destination"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	Amount      string `json:"amount"`
	Memo        string `json:"memo"`
	MemoType    string `json:"memo_type" example:"MEMO_TEXT"`
	Network     string `json:"network"`
	// Uri — payload SEP-0007, dán thẳng vào ví hoặc render thành QR.
	Uri string `json:"uri"`
}

// NormalizeAmount — ép amount về đúng dạng Stellar chấp nhận, hoặc từ chối thẳng.
//
// Không làm tròn hộ: một order 10.123456789 mà backend lặng lẽ ghi thành 10.1234568
// thì buyer trả đúng con số mình thấy vẫn bị matcher gọi là underpayment, và không ai
// truy ra được vì cả hai bên đều "đúng".
func NormalizeAmount(amount decimal.Decimal) (string, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return "", fmt.Errorf("amount phải lớn hơn 0")
	}
	if amount.GreaterThan(MaxAmount) {
		return "", fmt.Errorf("amount vượt trần int64 stroops của Stellar (%s)", MaxAmount)
	}
	if amount.Exponent() < -StellarDecimals {
		return "", fmt.Errorf("amount tối đa %d chữ số thập phân, nhận %s", StellarDecimals, amount)
	}
	return amount.StringFixed(StellarDecimals), nil
}

// ValidMemo — memo phải khác rỗng và vừa giới hạn MEMO_TEXT (tính theo BYTE, không
// phải rune: Horizon đếm byte).
func ValidMemo(memo string) bool {
	return memo != "" && len(memo) <= MemoTextMaxBytes
}

// BuildInstruction — dựng lệnh thanh toán từ config + amount + memo của order.
func (c *StellarConfig) BuildInstruction(amount decimal.Decimal, memo string) (*PaymentInstruction, error) {
	amountStr, err := NormalizeAmount(amount)
	if err != nil {
		return nil, err
	}
	if !ValidMemo(memo) {
		return nil, fmt.Errorf("memo rỗng hoặc dài quá %d byte: %q", MemoTextMaxBytes, memo)
	}

	q := url.Values{}
	q.Set("destination", c.DestinationAccount)
	q.Set("amount", amountStr)
	q.Set("asset_code", c.AssetCode)
	q.Set("asset_issuer", c.AssetIssuer)
	q.Set("memo", memo)
	q.Set("memo_type", "MEMO_TEXT")
	q.Set("network_passphrase", c.NetworkPassphrase)

	return &PaymentInstruction{
		Destination: c.DestinationAccount,
		AssetCode:   c.AssetCode,
		AssetIssuer: c.AssetIssuer,
		Amount:      amountStr,
		Memo:        memo,
		MemoType:    "MEMO_TEXT",
		Network:     c.NetworkPassphrase,
		Uri:         "web+stellar:pay?" + q.Encode(),
	}, nil
}

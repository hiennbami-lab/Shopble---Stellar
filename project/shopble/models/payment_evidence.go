package models

import "github.com/shopspring/decimal"

type Verdict string

const (
	VerdictMatched   Verdict = "matched"
	VerdictRejected  Verdict = "rejected"
	VerdictDuplicate Verdict = "duplicate"
)

// PaymentEvidence — bằng chứng thô đọc từ ledger cho MỘT payment operation, cộng với
// verdict mà matcher trả về. Đây là thứ reviewer đối chiếu với Stellar Expert, nên nó
// lưu nguyên giá trị quan sát được, không lưu giá trị đã chuẩn hoá theo order.
type PaymentEvidence struct {
	Id string `gorm:"column:id;primaryKey;type:varchar(32)"`
	// OpId — id của payment operation trên Horizon. Unique: đây là thứ duy nhất làm
	// ingestion replay-safe. Watcher restart và stream lại từ cursor cũ là chuyện bình
	// thường; không có ràng buộc này thì mỗi lần restart lại đếm trùng payment.
	OpId               string          `gorm:"column:op_id;type:varchar(40);uniqueIndex;not null"`
	PagingToken        string          `gorm:"column:paging_token;type:varchar(40);not null"`
	TxHash             string          `gorm:"column:tx_hash;type:varchar(64);index;not null"`
	SourceAccount      string          `gorm:"column:source_account;type:varchar(56);not null"`
	DestinationAccount string          `gorm:"column:destination_account;type:varchar(56);not null"`
	AssetCode          string          `gorm:"column:asset_code;type:varchar(12);not null"`
	AssetIssuer        string          `gorm:"column:asset_issuer;type:varchar(56)"`
	Amount             decimal.Decimal `gorm:"column:amount;type:numeric(23,7);not null"`
	Memo               string          `gorm:"column:memo;type:varchar(64)"`
	LedgerCloseAt      int64           `gorm:"column:ledger_close_at;not null"`
	// CapturedAt — lúc backend nhìn thấy payment. Hiệu với LedgerCloseAt là detection
	// latency mà SOW bắt báo cáo cho từng transaction.
	CapturedAt   int64        `gorm:"column:captured_at;autoCreateTime;index"`
	OrderId      string       `gorm:"column:order_id;type:varchar(32);index"`
	Verdict      Verdict      `gorm:"column:verdict;type:varchar(16);index"`
	RejectReason RejectReason `gorm:"column:reject_reason;type:varchar(32)"`
}

func (PaymentEvidence) TableName() string { return "shopble_payment_evidence" }

// WatcherCursor — paging token cuối cùng đã xử lý xong, lưu theo tên stream.
// Chỉ ghi SAU khi evidence đã commit, nếu không một lần crash đúng khe sẽ nhảy qua
// payment chưa kịp lưu và không bao giờ đọc lại nó nữa.
type WatcherCursor struct {
	Stream    string `gorm:"column:stream;primaryKey;type:varchar(64)"`
	Cursor    string `gorm:"column:cursor;type:varchar(40);not null"`
	UpdatedAt int64  `gorm:"column:updated_at;autoUpdateTime"`
}

func (WatcherCursor) TableName() string { return "shopble_watcher_cursor" }

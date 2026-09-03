package watcher

import (
	"context"
	"log"
	"time"

	"shopble/database"
	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"

	"github.com/segmentio/ksuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Watcher — Deliverable 2. Poll payment operations của destination account từ Horizon,
// ghi evidence thô, chạy matcher, và đẩy order qua state machine. Là comrunner.Service
// nên tắt sạch khi nhận SIGTERM.
//
// Replay-safe nhờ PaymentEvidence.op_id unique: restart và stream lại từ cursor cũ chỉ
// tạo ra INSERT bị nuốt bởi ON CONFLICT DO NOTHING, không đếm trùng payment.
type Watcher struct {
	cfg      *libstellar.StellarConfig
	stream   string
	interval time.Duration
	limit    int

	ctx    context.Context
	cancel context.CancelFunc
}

func New(stream string, interval time.Duration) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		cfg:      libstellar.GetStellarConfig(),
		stream:   stream,
		interval: interval,
		limit:    200,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (w *Watcher) Name() string { return "watcher" }
func (w *Watcher) BeforeStop()  {}

func (w *Watcher) Stop(ctx context.Context) error {
	w.cancel()
	return nil
}

// Start — vòng poll cho tới khi Stop() huỷ ctx. Trang đầy thì fetch tiếp ngay (đang
// bắt kịp lịch sử); trang rỗng thì ngủ interval (đã bắt kịp tip của ledger).
func (w *Watcher) Start() error {
	log.Printf("watcher: stream=%q dest=%s interval=%s", w.stream, w.cfg.DestinationAccount, w.interval)
	for {
		select {
		case <-w.ctx.Done():
			return nil
		default:
		}
		n, err := w.pollOnce()
		if err != nil {
			log.Printf("watcher: poll error: %v", err)
			w.sleep()
			continue
		}
		if n < w.limit {
			w.sleep()
		}
	}
}

func (w *Watcher) sleep() {
	select {
	case <-w.ctx.Done():
	case <-time.After(w.interval):
	}
}

// pollOnce — đọc một trang từ cursor hiện tại, xử lý từng record, trả số record đọc được.
func (w *Watcher) pollOnce() (int, error) {
	db := database.GetDb().DB

	cursor := w.loadCursor(db)
	payments, err := w.cfg.FetchPayments(w.ctx, cursor, w.limit)
	if err != nil {
		return 0, err
	}
	for i := range payments {
		if err := w.process(db, payments[i]); err != nil {
			// Không advance cursor: lần poll sau đọc lại đúng record này. Trả lỗi để
			// vòng ngoài ngủ rồi thử lại, thay vì nhảy qua một payment chưa xử lý xong.
			return i, err
		}
	}
	return len(payments), nil
}

func (w *Watcher) loadCursor(db *gorm.DB) string {
	var wc models.WatcherCursor
	if err := db.Where("stream = ?", w.stream).Take(&wc).Error; err != nil {
		return "" // chưa có cursor → đọc từ đầu lịch sử account
	}
	return wc.Cursor
}

// process — một payment: evidence + verdict + đổi trạng thái order + advance cursor,
// tất cả trong MỘT transaction. Evidence chèn ON CONFLICT DO NOTHING theo op_id; chỉ
// khi chèn MỚI thật sự (RowsAffected>0) mới chạm tới order.
func (w *Watcher) process(db *gorm.DB, p libstellar.HorizonPayment) error {
	amount, err := decimal.NewFromString(p.Amount)
	if err != nil {
		amount = decimal.Zero
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var order *models.OrderIntent
		if p.Memo != "" {
			var row models.OrderIntent
			if err := tx.Where("memo = ?", p.Memo).Take(&row).Error; err == nil {
				order = &row
			}
		}

		verdict, reason := Match(Observed{
			Source:      p.From,
			Destination: p.To,
			AssetCode:   p.AssetCode,
			AssetIssuer: p.AssetIssuer,
			Amount:      amount,
			Memo:        p.Memo,
		}, order)

		orderId := ""
		if order != nil {
			orderId = order.Id
		}
		ev := models.PaymentEvidence{
			Id:                 ksuid.New().String(),
			OpId:               p.Id,
			PagingToken:        p.PagingToken,
			TxHash:             p.TransactionHash,
			SourceAccount:      p.From,
			DestinationAccount: p.To,
			AssetCode:          p.AssetCode,
			AssetIssuer:        p.AssetIssuer,
			Amount:             amount,
			Memo:               p.Memo,
			LedgerCloseAt:      p.LedgerCloseAt,
			OrderId:            orderId,
			Verdict:            verdict,
			RejectReason:       reason,
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ev)
		if res.Error != nil {
			return res.Error
		}

		// RowsAffected==0 → op_id đã xử lý trước đó (replay). Đừng chạm order, chỉ advance.
		if res.RowsAffected > 0 && order != nil && verdict != models.VerdictDuplicate {
			// awaiting_payment → payment_detected → validated|rejected. Đồng bộ nên chốt
			// thẳng terminal; guard status=awaiting_payment giữ idempotent với payment sau.
			// ponytail: không lưu payment_detected trung gian; matching đồng bộ nên nó thoáng qua.
			terminal := Terminal(verdict)
			upd := tx.Model(&models.OrderIntent{}).
				Where("id = ? AND status = ?", order.Id, models.OrderStatusAwaitingPayment).
				Updates(map[string]any{"status": terminal, "reject_reason": reason})
			if upd.Error != nil {
				return upd.Error
			}
			log.Printf("watcher: order=%s memo=%s verdict=%s reason=%q op=%s", order.Id, p.Memo, verdict, reason, p.Id)
		}

		// Cursor chỉ ghi trong txn này, cùng lúc evidence commit (model comment yêu cầu).
		return tx.Save(&models.WatcherCursor{Stream: w.stream, Cursor: p.PagingToken}).Error
	})
}

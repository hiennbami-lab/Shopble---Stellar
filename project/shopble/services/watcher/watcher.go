package watcher

import (
	"context"
	"fmt"
	"log"
	"time"

	"shopble/database"
	"shopble/project/shopble/lib/libsoroban"
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
	chain    *libsoroban.Client
	stream   string
	interval time.Duration
	limit    int

	ctx    context.Context
	cancel context.CancelFunc
}

// New — chainWrites=false tắt hẳn việc ghi lên contract.
//
// Tắt phải là một lựa chọn NÓI RA. Nếu thiếu secret mà tự động bỏ qua ghi chain, sẽ có
// lúc chạy thật với order được validate trong Postgres nhưng không bao giờ lên chain, và
// không ai nhận ra cho tới lúc đi tìm bằng chứng on-chain. Nên: mặc định chết lúc boot,
// còn muốn chạy không chain thì phải tự tay bật cờ.
func New(stream string, interval time.Duration, chainWrites bool) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := libstellar.GetStellarConfig()

	var chain *libsoroban.Client
	if !chainWrites {
		log.Printf("watcher: --no-chain, KHÔNG ghi verdict lên contract")
	} else {
		// Contract chưa deploy thì chain == nil và watcher chạy bình thường. Nhưng
		// contract_id đã set mà secret sai/thiếu là sai cấu hình — phải chết lúc boot.
		c, err := libsoroban.New(cfg)
		if err != nil {
			panic(fmt.Errorf("watcher: soroban config: %w\n"+
				"đặt secret vào $%s, hoặc chạy với --no-chain nếu cố ý không ghi on-chain",
				err, cfg.OperatorSecretEnvVar))
		}
		chain = c
	}
	return &Watcher{
		cfg:      cfg,
		chain:    chain,
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
//
// Ghi on-chain nằm NGOÀI transaction: mỗi lần gọi contract mất vài giây chờ ledger,
// giữ transaction mở suốt thời gian đó sẽ khoá row và cạn connection pool.
func (w *Watcher) process(db *gorm.DB, p libstellar.HorizonPayment) error {
	amount, err := decimal.NewFromString(p.Amount)
	if err != nil {
		amount = decimal.Zero
	}

	// Set khi order vừa được chốt trong txn dưới đây → cần ghi lên chain sau khi commit.
	var (
		settled *models.OrderIntent
		verdict models.Verdict
		reason  models.RejectReason
	)

	err = db.Transaction(func(tx *gorm.DB) error {
		var order *models.OrderIntent
		if p.Memo != "" {
			var row models.OrderIntent
			if err := tx.Where("memo = ?", p.Memo).Take(&row).Error; err == nil {
				order = &row
			}
		}

		v, r := Match(Observed{
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
			Verdict:            v,
			RejectReason:       r,
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ev)
		if res.Error != nil {
			return res.Error
		}

		// RowsAffected==0 → op_id đã xử lý trước đó (replay). Đừng chạm order, chỉ advance.
		if res.RowsAffected > 0 && order != nil && v != models.VerdictDuplicate {
			// Đi ĐÚNG hai bước của state machine: awaiting_payment → payment_detected
			// → validated|rejected. Bước giữa không còn là hình thức: contract Soroban
			// từ chối nhảy thẳng tới validated, nên thứ tự này phải có thật.
			det := tx.Model(&models.OrderIntent{}).
				Where("id = ? AND status = ?", order.Id, models.OrderStatusAwaitingPayment).
				Update("status", models.OrderStatusPaymentDetected)
			if det.Error != nil {
				return det.Error
			}
			// RowsAffected==0 → payment khác đã chốt order này trước. Không ghi đè.
			if det.RowsAffected > 0 {
				terminal := Terminal(v)
				upd := tx.Model(&models.OrderIntent{}).
					Where("id = ? AND status = ?", order.Id, models.OrderStatusPaymentDetected).
					Updates(map[string]any{"status": terminal, "reject_reason": r})
				if upd.Error != nil {
					return upd.Error
				}
				settled, verdict, reason = order, v, r
				log.Printf("watcher: order=%s memo=%s verdict=%s reason=%q op=%s", order.Id, p.Memo, v, r, p.Id)
			}
		}

		// Cursor chỉ ghi trong txn này, cùng lúc evidence commit (model comment yêu cầu).
		return tx.Save(&models.WatcherCursor{Stream: w.stream, Cursor: p.PagingToken}).Error
	})
	if err != nil {
		return err
	}

	if settled != nil {
		w.writeChain(settled, verdict, reason)
	}
	return nil
}

// writeChain — ghi verdict lên Soroban contract (Deliverable 3).
//
// Lỗi ở đây KHÔNG làm hỏng việc xử lý payment: Postgres đã là bản ghi chuẩn và
// evidence đã commit. Chain là bản sao kiểm chứng được, nên hỏng thì log rồi đi tiếp,
// không kéo cả watcher dừng lại.
func (w *Watcher) writeChain(order *models.OrderIntent, verdict models.Verdict, reason models.RejectReason) {
	if w.chain == nil {
		return
	}
	ctx, cancel := context.WithTimeout(w.ctx, 3*time.Minute)
	defer cancel()

	// Đăng ký order nếu chưa có trên chain. Gọi lại trên order đã tồn tại sẽ lỗi
	// AlreadyExists — vô hại, vì set_status ngay dưới mới là bước bắt buộc thành công.
	if _, err := w.chain.CreateOrder(ctx, order); err != nil {
		log.Printf("watcher: chain create_order order=%s: %v", order.Id, err)
	}
	if _, err := w.chain.SetStatus(ctx, order.Id, models.OrderStatusPaymentDetected, ""); err != nil {
		log.Printf("watcher: chain payment_detected order=%s: %v", order.Id, err)
		return
	}
	terminal := Terminal(verdict)
	hash, err := w.chain.SetStatus(ctx, order.Id, terminal, reason)
	if err != nil {
		log.Printf("watcher: chain %s order=%s: %v", terminal, order.Id, err)
		return
	}
	if err := database.GetDb().DB.Model(&models.OrderIntent{}).
		Where("id = ?", order.Id).
		Update("on_chain_tx_hash", hash).Error; err != nil {
		log.Printf("watcher: lưu on_chain_tx_hash order=%s: %v", order.Id, err)
	}
	log.Printf("watcher: order=%s on-chain %s tx=%s", order.Id, terminal, hash)
}

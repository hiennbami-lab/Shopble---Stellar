package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"
)

// Report — sinh bảng kết quả mà SOW Deliverable 1/2/3 bắt nộp, thẳng từ dữ liệu đã lưu.
//
// Lắp tay 25 dòng bảng thì sai một dòng cũng không ai phát hiện, và bảng chép tay không
// còn là bằng chứng nữa. Ở đây bảng đọc cùng nguồn với API, cộng thêm phần coverage nói
// thẳng campaign còn thiếu case nào — thứ mà nhìn bảng bằng mắt sẽ bỏ sót.
//
// Hàm thuần: nhận sẵn dữ liệu, không chạm DB. Chỗ gọi (cmd/report.go) lo phần query.

// Ngưỡng SOW: 10 order intent run trên ít nhất 3 ví, 15 transaction watcher.
const (
	TargetOrders  = 10
	TargetWallets = 3
	TargetTxs     = 15
)

// Coverage — campaign đã đủ theo SOW chưa. Trả về dữ liệu chứ không in ra, để test
// khẳng định được từng con số.
type Coverage struct {
	Orders          int
	DistinctWallets int
	Evidence        int
	Verdicts        map[models.Verdict]int
	ReasonsSeen     []models.RejectReason
	ReasonsMissing  []models.RejectReason
	// OnChainMissing — order đã chốt trong Postgres nhưng chưa có transaction on-chain.
	// Mỗi id ở đây là một lỗ trong bằng chứng Deliverable 3.
	OnChainMissing []string
}

func Summarize(orders []models.OrderIntent, evidence []models.PaymentEvidence) Coverage {
	c := Coverage{
		Orders:   len(orders),
		Evidence: len(evidence),
		Verdicts: map[models.Verdict]int{},
	}

	wallets := map[string]bool{}
	for i := range orders {
		o := &orders[i]
		wallets[o.BuyerWallet] = true
		if models.OrderFinal(o.Status) && o.OnChainTxHash == "" {
			c.OnChainMissing = append(c.OnChainMissing, o.Id)
		}
	}
	c.DistinctWallets = len(wallets)

	seen := map[models.RejectReason]bool{}
	for i := range evidence {
		e := &evidence[i]
		c.Verdicts[e.Verdict]++
		if e.RejectReason != "" {
			seen[e.RejectReason] = true
		}
	}
	// Đi theo AllRejectReasons chứ không theo map: thứ tự phải ổn định giữa các lần chạy,
	// nếu không diff giữa hai bản báo cáo toàn là nhiễu.
	for _, r := range models.AllRejectReasons {
		if seen[r] {
			c.ReasonsSeen = append(c.ReasonsSeen, r)
		} else {
			c.ReasonsMissing = append(c.ReasonsMissing, r)
		}
	}
	sort.Strings(c.OnChainMissing)
	return c
}

// Markdown — toàn bộ evidence pack dạng markdown, dán thẳng vào packet nộp.
func Markdown(cfg *libstellar.StellarConfig, orders []models.OrderIntent, evidence []models.PaymentEvidence) string {
	cov := Summarize(orders, evidence)
	var b strings.Builder

	fmt.Fprintf(&b, "# Shopble — Phase 1 evidence pack\n\n")
	fmt.Fprintf(&b, "Sinh lúc %s bằng `shopble report`.\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "| | |\n|---|---|\n")
	fmt.Fprintf(&b, "| Network | `%s` |\n", cfg.NetworkPassphrase)
	fmt.Fprintf(&b, "| Asset | `%s` / `%s` |\n", cfg.AssetCode, cfg.AssetIssuer)
	fmt.Fprintf(&b, "| Destination | `%s` |\n", cfg.DestinationAccount)
	fmt.Fprintf(&b, "| Order-status contract | %s |\n\n", codeOrDash(cfg.ContractId))

	writeCoverage(&b, cov)
	writeOrders(&b, cfg, orders)
	writeTransactions(&b, evidence)
	writeChain(&b, cfg, orders)

	return b.String()
}

func writeCoverage(b *strings.Builder, c Coverage) {
	fmt.Fprintf(b, "## Coverage\n\n")
	fmt.Fprintf(b, "| Yêu cầu SOW | Có | Cần | |\n|---|---|---|---|\n")
	fmt.Fprintf(b, "| Order intent run (D1) | %d | %d | %s |\n", c.Orders, TargetOrders, mark(c.Orders >= TargetOrders))
	fmt.Fprintf(b, "| Ví buyer khác nhau (D1) | %d | %d | %s |\n", c.DistinctWallets, TargetWallets, mark(c.DistinctWallets >= TargetWallets))
	fmt.Fprintf(b, "| Transaction watcher (D2) | %d | %d | %s |\n", c.Evidence, TargetTxs, mark(c.Evidence >= TargetTxs))
	fmt.Fprintf(b, "| Lý do từ chối có mặt (D2) | %d | %d | %s |\n\n", len(c.ReasonsSeen), len(models.AllRejectReasons), mark(len(c.ReasonsMissing) == 0))

	fmt.Fprintf(b, "Verdict: matched %d · rejected %d · duplicate %d\n\n",
		c.Verdicts[models.VerdictMatched], c.Verdicts[models.VerdictRejected], c.Verdicts[models.VerdictDuplicate])

	if len(c.ReasonsMissing) > 0 {
		fmt.Fprintf(b, "**Còn thiếu case cho:** %s — campaign chưa phủ hết 5 lý do từ chối.\n\n", joinReasons(c.ReasonsMissing))
	}
	if len(c.OnChainMissing) > 0 {
		fmt.Fprintf(b, "**Chốt trong Postgres nhưng chưa lên chain:** `%s`. Watcher sẽ tự thử ghi lại ở vòng poll sau; nếu vẫn trống thì đọc log để biết contract chặn ở đâu.\n\n",
			strings.Join(c.OnChainMissing, "`, `"))
	}
}

func writeOrders(b *strings.Builder, cfg *libstellar.StellarConfig, orders []models.OrderIntent) {
	fmt.Fprintf(b, "## D1 — Order intent runs\n\n")
	fmt.Fprintf(b, "7 field lưu theo SOW, cộng payment instruction sinh ra cho từng order.\n\n")
	fmt.Fprintf(b, "| # | Order id (= memo) | Product | Expected | Asset | Issuer | Destination | Buyer wallet | Status | Instruction |\n")
	fmt.Fprintf(b, "|---|---|---|---|---|---|---|---|---|---|\n")
	for i := range orders {
		o := &orders[i]
		uri := "—"
		if ins, err := cfg.BuildInstruction(o.ExpectedAmount, o.Memo); err == nil {
			uri = "`" + ins.Uri + "`"
		}
		fmt.Fprintf(b, "| %d | `%s` | %s | %s | `%s` | `%s` | `%s` | `%s` | %s | %s |\n",
			i+1, o.Id, o.ProductRef, o.ExpectedAmount.StringFixed(libstellar.StellarDecimals),
			o.AssetCode, short(o.AssetIssuer), short(o.DestinationAccount), short(o.BuyerWallet),
			statusText(o), uri)
	}
	fmt.Fprintf(b, "\n")
}

func writeTransactions(b *strings.Builder, evidence []models.PaymentEvidence) {
	fmt.Fprintf(b, "## D2 — Watcher transactions\n\n")
	fmt.Fprintf(b, "Cột **Expected** là ý định của test case, không suy ra được từ ledger — người chạy campaign tự điền. Mọi cột còn lại đọc từ dữ liệu đã lưu.\n\n")
	fmt.Fprintf(b, "| # | Tx | Source | Amount | Asset | Memo | Order | Expected | Verdict | Reason | Latency |\n")
	fmt.Fprintf(b, "|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i := range evidence {
		e := &evidence[i]
		// Backfill (payment có từ trước khi watcher chạy) cho ra latency là khoảng cách
		// tới quá khứ, không phải độ trễ phát hiện. Đánh dấu thay vì báo một con số sai.
		lat := e.CapturedAt - e.LedgerCloseAt
		latText := fmt.Sprintf("%ds", lat)
		if lat < 0 {
			latText = "0s"
		} else if lat > 300 {
			latText = fmt.Sprintf("%ds (backfill)", lat)
		}
		fmt.Fprintf(b, "| %d | [`%s`](%s) | `%s` | %s | `%s` | %s | %s | | %s | %s | %s |\n",
			i+1, short(e.TxHash), libstellar.ExplorerTxUrl(e.TxHash), short(e.SourceAccount),
			e.Amount.StringFixed(libstellar.StellarDecimals), e.AssetCode, code(e.Memo), code(e.OrderId),
			e.Verdict, dash(string(e.RejectReason)), latText)
	}
	fmt.Fprintf(b, "\n")
}

func writeChain(b *strings.Builder, cfg *libstellar.StellarConfig, orders []models.OrderIntent) {
	fmt.Fprintf(b, "## D3 — On-chain status\n\n")
	fmt.Fprintf(b, "Contract: %s\n\n", codeOrDash(cfg.ContractId))
	fmt.Fprintf(b, "| Order | Trạng thái cuối | Reason | Transaction ghi verdict |\n|---|---|---|---|\n")
	n := 0
	for i := range orders {
		o := &orders[i]
		if !models.OrderFinal(o.Status) {
			continue
		}
		n++
		link := "— (chưa ghi được)"
		if o.OnChainTxHash != "" {
			link = fmt.Sprintf("[`%s`](%s)", short(o.OnChainTxHash), libstellar.ExplorerTxUrl(o.OnChainTxHash))
		}
		fmt.Fprintf(b, "| `%s` | %s | %s | %s |\n", o.Id, o.Status, dash(string(o.RejectReason)), link)
	}
	if n == 0 {
		fmt.Fprintf(b, "| — | — | — | chưa có order nào chốt |\n")
	}
	fmt.Fprintf(b, "\n")
}

func statusText(o *models.OrderIntent) string {
	if o.RejectReason != "" {
		return fmt.Sprintf("%s (%s)", o.Status, o.RejectReason)
	}
	return string(o.Status)
}

// short — rút gọn strkey/hash cho vừa bảng. Giá trị đầy đủ vẫn nằm sau link explorer
// và trong API, nên rút gọn ở đây không làm mất bằng chứng.
func short(s string) string {
	if len(s) <= 14 {
		return s
	}
	return s[:6] + "…" + s[len(s)-4:]
}

// code — giá trị dạng code, hoặc gạch ngang khi rỗng. Payment không memo là chuyện
// bình thường (backfill lịch sử account), in ra cặp backtick rỗng thì bảng khó đọc.
func code(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + s + "`"
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func codeOrDash(s string) string {
	if s == "" {
		return "— (chưa deploy)"
	}
	return "`" + s + "`"
}

func mark(ok bool) string {
	if ok {
		return "OK"
	}
	return "THIẾU"
}

func joinReasons(rs []models.RejectReason) string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, "`"+string(r)+"`")
	}
	return strings.Join(out, ", ")
}

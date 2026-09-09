package report

import (
	"strings"
	"testing"

	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"

	"github.com/shopspring/decimal"
)

func cfg() *libstellar.StellarConfig {
	return &libstellar.StellarConfig{
		NetworkPassphrase:  libstellar.TestnetPassphrase,
		AssetCode:          "USDC",
		AssetIssuer:        "GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5",
		DestinationAccount: "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ",
		ContractId:         "CCVC5G47V7EBMZNCRO2SBIJRD67RTMIZMOL75UEAIHCAF3HHXXNQNHCO",
	}
}

func TestSummarize(t *testing.T) {
	orders := []models.OrderIntent{
		{Id: "o1", BuyerWallet: "GA", Status: models.OrderStatusValidated, OnChainTxHash: "hash1"},
		{Id: "o2", BuyerWallet: "GA", Status: models.OrderStatusAwaitingPayment},
		// Chốt rồi mà không có tx hash — đúng cái lỗ bằng chứng D3 cần bị chỉ ra.
		{Id: "o3", BuyerWallet: "GB", Status: models.OrderStatusRejected, RejectReason: models.RejectUnderpayment},
	}
	evidence := []models.PaymentEvidence{
		{Verdict: models.VerdictMatched},
		{Verdict: models.VerdictRejected, RejectReason: models.RejectUnderpayment},
		{Verdict: models.VerdictRejected, RejectReason: models.RejectWrongAsset},
		{Verdict: models.VerdictDuplicate},
	}

	c := Summarize(orders, evidence)
	if c.Orders != 3 || c.DistinctWallets != 2 || c.Evidence != 4 {
		t.Fatalf("counts: %+v", c)
	}
	if c.Verdicts[models.VerdictMatched] != 1 || c.Verdicts[models.VerdictRejected] != 2 ||
		c.Verdicts[models.VerdictDuplicate] != 1 {
		t.Fatalf("verdicts: %+v", c.Verdicts)
	}
	if len(c.ReasonsSeen) != 2 || len(c.ReasonsMissing) != 3 {
		t.Fatalf("reasons seen=%v missing=%v", c.ReasonsSeen, c.ReasonsMissing)
	}
	// Ba lý do còn thiếu phải giữ thứ tự của AllRejectReasons.
	want := []models.RejectReason{
		models.RejectWrongDestination, models.RejectWrongBuyerWallet, models.RejectInvalidMemo,
	}
	for i := range want {
		if c.ReasonsMissing[i] != want[i] {
			t.Fatalf("missing[%d]=%s want %s", i, c.ReasonsMissing[i], want[i])
		}
	}
	if len(c.OnChainMissing) != 1 || c.OnChainMissing[0] != "o3" {
		t.Fatalf("on-chain missing: %v", c.OnChainMissing)
	}
}

func TestMarkdown(t *testing.T) {
	orders := []models.OrderIntent{{
		Id: "2ABc", ProductRef: "sku-1", ExpectedAmount: decimal.RequireFromString("25.5"),
		AssetCode: "USDC", AssetIssuer: cfg().AssetIssuer, DestinationAccount: cfg().DestinationAccount,
		Memo: "2ABc", BuyerWallet: "GBUYER", Status: models.OrderStatusValidated, OnChainTxHash: "abc123",
	}}
	evidence := []models.PaymentEvidence{{
		TxHash: "deadbeef", SourceAccount: "GBUYER", AssetCode: "USDC",
		Amount: decimal.RequireFromString("25.5"), Memo: "2ABc", OrderId: "2ABc",
		Verdict: models.VerdictMatched, LedgerCloseAt: 100, CapturedAt: 104,
	}}

	out := Markdown(cfg(), orders, evidence)
	for _, want := range []string{
		"web+stellar:pay?", // instruction sinh lại được cho từng order
		"https://stellar.expert/explorer/testnet/tx/deadbeef", // link explorer bấm được
		"4s",                  // detection latency
		"1 | 10 | THIẾU",      // coverage nói thẳng còn thiếu
		"`wrong_destination`", // và thiếu ở đâu
		"CCVC5G47",            // contract id trong mục D3
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("báo cáo thiếu %q\n%s", want, out)
		}
	}
	// Order chưa chốt không được xuất hiện ở bảng D3.
	if strings.Contains(out, "chưa có order nào chốt") {
		t.Fatal("order validated phải có dòng trong bảng D3")
	}
}

package watcher

import (
	"testing"

	"shopble/project/shopble/models"

	"github.com/shopspring/decimal"
)

func baseOrder() *models.OrderIntent {
	return &models.OrderIntent{
		Id:                 "ord1",
		ExpectedAmount:     decimal.RequireFromString("25.5"),
		AssetCode:          "USDC",
		AssetIssuer:        "GISSUER",
		DestinationAccount: "GDEST",
		BuyerWallet:        "GBUYER",
		Memo:               "ord1",
		Status:             models.OrderStatusAwaitingPayment,
	}
}

func good() Observed {
	return Observed{
		Source:      "GBUYER",
		Destination: "GDEST",
		AssetCode:   "USDC",
		AssetIssuer: "GISSUER",
		Amount:      decimal.RequireFromString("25.5"),
		Memo:        "ord1",
	}
}

func TestMatch(t *testing.T) {
	type tc struct {
		name    string
		mut     func(o *Observed, ord *models.OrderIntent)
		verdict models.Verdict
		reason  models.RejectReason
	}
	cases := []tc{
		{"matched exact", nil, models.VerdictMatched, ""},
		{"overpay matched", func(o *Observed, _ *models.OrderIntent) {
			o.Amount = decimal.RequireFromString("100")
		}, models.VerdictMatched, ""},
		{"underpay one stroop", func(o *Observed, _ *models.OrderIntent) {
			o.Amount = decimal.RequireFromString("25.4999999")
		}, models.VerdictRejected, models.RejectUnderpayment},
		{"wrong destination", func(o *Observed, _ *models.OrderIntent) {
			o.Destination = "GOTHER"
		}, models.VerdictRejected, models.RejectWrongDestination},
		{"wrong asset code", func(o *Observed, _ *models.OrderIntent) {
			o.AssetCode = "USDT"
		}, models.VerdictRejected, models.RejectWrongAsset},
		{"wrong asset issuer", func(o *Observed, _ *models.OrderIntent) {
			o.AssetIssuer = "GFAKE"
		}, models.VerdictRejected, models.RejectWrongAsset},
		{"wrong buyer wallet", func(o *Observed, _ *models.OrderIntent) {
			o.Source = "GATTACKER"
		}, models.VerdictRejected, models.RejectWrongBuyerWallet},
		{"final order duplicate", func(_ *Observed, ord *models.OrderIntent) {
			ord.Status = models.OrderStatusValidated
		}, models.VerdictDuplicate, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, ord := good(), baseOrder()
			if c.mut != nil {
				c.mut(&o, ord)
			}
			v, r := Match(o, ord)
			if v != c.verdict || r != c.reason {
				t.Fatalf("got (%s,%s), want (%s,%s)", v, r, c.verdict, c.reason)
			}
		})
	}

	// nil order (memo khớp không ai) → invalid_memo.
	if v, r := Match(good(), nil); v != models.VerdictRejected || r != models.RejectInvalidMemo {
		t.Fatalf("nil order: got (%s,%s), want rejected/invalid_memo", v, r)
	}

	// Priority: destination sai được ưu tiên hơn asset/amount sai.
	o, ord := good(), baseOrder()
	o.Destination, o.AssetCode, o.Amount = "GOTHER", "USDT", decimal.Zero
	if _, r := Match(o, ord); r != models.RejectWrongDestination {
		t.Fatalf("priority: want wrong_destination first, got %s", r)
	}
}

func TestTerminal(t *testing.T) {
	if Terminal(models.VerdictMatched) != models.OrderStatusValidated {
		t.Fatal("matched must map to validated")
	}
	if Terminal(models.VerdictRejected) != models.OrderStatusRejected {
		t.Fatal("rejected must map to rejected")
	}
}

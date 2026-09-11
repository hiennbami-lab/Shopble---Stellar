//go:build shopble

package libstellar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Horizon /payments trộn nhiều loại operation. Chỉ những loại thực sự chuyển tiền
// vào destination mới được vào watcher.
const paymentsPage = `{"_embedded":{"records":[
{"id":"1","paging_token":"1","type":"create_account","transaction_hash":"h0","account":"GBUYER","starting_balance":"10","created_at":"2026-09-01T00:00:00Z"},
{"id":"2","paging_token":"2","type":"payment","transaction_hash":"h1","from":"GBUYER","to":"GDEST","asset_type":"credit_alphanum4","asset_code":"USDC","asset_issuer":"GISSUER","amount":"25.0000000","created_at":"2026-09-01T00:01:00Z","transaction":{"memo":"order1","memo_type":"text"}},
{"id":"3","paging_token":"3","type":"path_payment_strict_receive","transaction_hash":"h2","from":"GBUYER","to":"GDEST","asset_type":"credit_alphanum4","asset_code":"USDC","asset_issuer":"GISSUER","amount":"30.0000000","source_asset_type":"native","source_amount":"999.0000000","created_at":"2026-09-01T00:02:00Z","transaction":{"memo":"order2","memo_type":"text"}},
{"id":"4","paging_token":"4","type":"path_payment_strict_send","transaction_hash":"h3","from":"GBUYER","to":"GDEST","asset_type":"native","amount":"5.0000000","created_at":"2026-09-01T00:03:00Z","transaction":{"memo":"deadbeef","memo_type":"hash"}}
]}}`

func TestFetchPaymentsAcceptsPathPayments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(paymentsPage))
	}))
	defer srv.Close()

	got, err := (&StellarConfig{HorizonUrl: srv.URL, DestinationAccount: "GDEST"}).
		FetchPayments(context.Background(), "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 payments (create_account dropped), got %d", len(got))
	}

	// Path payment: amount/asset là thứ destination nhận, không phải source_amount.
	p := got[1]
	if p.Amount != "30.0000000" || p.AssetCode != "USDC" || p.Memo != "order2" {
		t.Errorf("path_payment_strict_receive: got %+v", p)
	}
	// Native + memo không phải text vẫn phải vào, để matcher trả wrong_asset/invalid_memo.
	if got[2].AssetCode != "XLM" || got[2].Memo != "" {
		t.Errorf("native path payment: got %+v", got[2])
	}
	if got[0].LedgerCloseAt == 0 {
		t.Error("created_at not parsed")
	}
}

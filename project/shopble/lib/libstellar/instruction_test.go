package libstellar

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNormalizeAmount(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "10", want: "10.0000000"},
		{in: "0.1234567", want: "0.1234567"},
		{in: "0.12345678", wantErr: true}, // 8 chữ số thập phân
		{in: "0", wantErr: true},
		{in: "-1", wantErr: true},
		{in: "922337203685.4775808", wantErr: true}, // tràn int64 stroops
	}
	for _, tc := range cases {
		got, err := NormalizeAmount(decimal.RequireFromString(tc.in))
		if tc.wantErr {
			if err == nil {
				t.Errorf("NormalizeAmount(%s) = %q, muốn lỗi", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeAmount(%s) lỗi: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizeAmount(%s) = %q, muốn %q", tc.in, got, tc.want)
		}
	}
}

func TestValidMemo(t *testing.T) {
	if ValidMemo("") {
		t.Error("memo rỗng phải invalid")
	}
	if !ValidMemo(strings.Repeat("a", MemoTextMaxBytes)) {
		t.Error("memo đúng 28 byte phải valid")
	}
	if ValidMemo(strings.Repeat("a", MemoTextMaxBytes+1)) {
		t.Error("memo 29 byte phải invalid")
	}
	// 28 rune tiếng Việt = 56 byte: đếm theo rune sẽ lọt, Horizon vẫn từ chối.
	if ValidMemo(strings.Repeat("ố", MemoTextMaxBytes)) {
		t.Error("memo phải đếm theo byte, không phải rune")
	}
}

func TestBuildInstruction(t *testing.T) {
	c := &StellarConfig{
		DestinationAccount: "GDEST",
		AssetCode:          "USDC",
		AssetIssuer:        "GISSUER",
		NetworkPassphrase:  TestnetPassphrase,
	}
	got, err := c.BuildInstruction(decimal.RequireFromString("25.5"), "order-1")
	if err != nil {
		t.Fatalf("BuildInstruction lỗi: %v", err)
	}
	if got.Amount != "25.5000000" {
		t.Errorf("Amount = %q, muốn 25.5000000", got.Amount)
	}
	for _, want := range []string{"destination=GDEST", "asset_code=USDC", "memo=order-1", "memo_type=MEMO_TEXT"} {
		if !strings.Contains(got.Uri, want) {
			t.Errorf("Uri thiếu %q: %s", want, got.Uri)
		}
	}
	if !strings.HasPrefix(got.Uri, "web+stellar:pay?") {
		t.Errorf("Uri sai scheme SEP-0007: %s", got.Uri)
	}
	if _, err := c.BuildInstruction(decimal.RequireFromString("25.5"), ""); err == nil {
		t.Error("memo rỗng phải bị từ chối")
	}
}

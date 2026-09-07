package libsoroban

import (
	"fmt"

	"shopble/project/shopble/models"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

// Chuyển giá trị Go sang ScVal của Soroban. Encoding của enum #[contracttype] là
// một vec: phần tử đầu là symbol tên variant, các phần tử sau là payload. Sai chỗ
// này thì contract panic ở host chứ không báo lỗi kiểu rõ ràng, nên giữ nó một chỗ.

func scSymbol(s string) xdr.ScVal {
	sym := xdr.ScSymbol(s)
	return xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &sym}
}

func scString(s string) xdr.ScVal {
	str := xdr.ScString(s)
	return xdr.ScVal{Type: xdr.ScValTypeScvString, Str: &str}
}

func scBytes(b []byte) xdr.ScVal {
	v := xdr.ScBytes(b)
	return xdr.ScVal{Type: xdr.ScValTypeScvBytes, Bytes: &v}
}

// scI128 — amount gửi xuống contract là stroops (int64), luôn không âm ở scope này.
func scI128(v int64) xdr.ScVal {
	hi := xdr.Int64(0)
	if v < 0 {
		hi = -1
	}
	parts := xdr.Int128Parts{Hi: hi, Lo: xdr.Uint64(uint64(v))}
	return xdr.ScVal{Type: xdr.ScValTypeScvI128, I128: &parts}
}

func scAccount(addr string) (xdr.ScVal, error) {
	aid, err := xdr.AddressToAccountId(addr)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("địa chỉ %q không hợp lệ: %w", addr, err)
	}
	a := xdr.ScAddress{Type: xdr.ScAddressTypeScAddressTypeAccount, AccountId: &aid}
	return xdr.ScVal{Type: xdr.ScValTypeScvAddress, Address: &a}, nil
}

func scEnum(variant string, payload ...xdr.ScVal) xdr.ScVal {
	vec := xdr.ScVec(append([]xdr.ScVal{scSymbol(variant)}, payload...))
	p := &vec
	return xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &p}
}

func contractAddress(contractId string) (xdr.ScAddress, error) {
	raw, err := strkey.Decode(strkey.VersionByteContract, contractId)
	if err != nil {
		return xdr.ScAddress{}, fmt.Errorf("contract_id %q không phải strkey C...: %w", contractId, err)
	}
	var id xdr.ContractId
	copy(id[:], raw)
	return xdr.ScAddress{Type: xdr.ScAddressTypeScAddressTypeContract, ContractId: &id}, nil
}

// rejectVariant — tên variant Rust cho từng lý do từ chối. Enum đóng cả hai phía:
// thêm lý do mới ở Go mà quên contract thì lỗi phải nổ ở đây, không phải trên chain.
func rejectVariant(r models.RejectReason) (string, error) {
	switch r {
	case models.RejectUnderpayment:
		return "Underpayment", nil
	case models.RejectWrongAsset:
		return "WrongAsset", nil
	case models.RejectWrongDestination:
		return "WrongDestination", nil
	case models.RejectWrongBuyerWallet:
		return "WrongBuyerWallet", nil
	case models.RejectInvalidMemo:
		return "InvalidMemo", nil
	}
	return "", fmt.Errorf("reject reason không có tên: %q", r)
}

// statusScVal — Status của contract. `rejected` bắt buộc kèm lý do có tên.
func statusScVal(s models.OrderStatus, reason models.RejectReason) (xdr.ScVal, error) {
	switch s {
	case models.OrderStatusAwaitingPayment:
		return scEnum("AwaitingPayment"), nil
	case models.OrderStatusPaymentDetected:
		return scEnum("PaymentDetected"), nil
	case models.OrderStatusValidated:
		return scEnum("Validated"), nil
	case models.OrderStatusRejected:
		v, err := rejectVariant(reason)
		if err != nil {
			return xdr.ScVal{}, err
		}
		return scEnum("Rejected", scEnum(v)), nil
	}
	return xdr.ScVal{}, fmt.Errorf("status không hợp lệ: %q", s)
}

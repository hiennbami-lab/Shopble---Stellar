package libsoroban

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"

	rpcclient "github.com/stellar/go/clients/rpcclient"
	"github.com/stellar/go/keypair"
	protocol "github.com/stellar/go/protocols/rpc"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

// Client — ghi verdict của matcher lên Soroban order-status contract (SOW Deliverable 3).
//
// Mọi ghi đều do operator ký. Secret key nằm trong biến môi trường mà config trỏ tới,
// KHÔNG nằm trong file config.
type Client struct {
	rpc        *rpcclient.Client
	kp         *keypair.Full
	contract   xdr.ScAddress
	passphrase string
}

// New — trả (nil, nil) khi contract_id chưa set: contract chưa deploy thì backend vẫn
// chạy bình thường, chỉ không ghi on-chain. Chưa deploy không phải là lỗi cấu hình.
func New(cfg *libstellar.StellarConfig) (*Client, error) {
	if cfg.ContractId == "" {
		return nil, nil
	}
	envVar := cfg.OperatorSecretEnvVar
	if envVar == "" {
		return nil, fmt.Errorf("STELLAR_CONFIG.operator_secret_env_var bắt buộc khi contract_id đã set")
	}
	secret := os.Getenv(envVar)
	if secret == "" {
		return nil, fmt.Errorf("biến môi trường %s rỗng — cần secret key S... của operator", envVar)
	}
	kp, err := keypair.ParseFull(secret)
	if err != nil {
		return nil, fmt.Errorf("%s không phải secret key hợp lệ: %w", envVar, err)
	}
	addr, err := contractAddress(cfg.ContractId)
	if err != nil {
		return nil, err
	}
	return &Client{
		rpc:        rpcclient.NewClient(cfg.SorobanRpcUrl, nil),
		kp:         kp,
		contract:   addr,
		passphrase: cfg.NetworkPassphrase,
	}, nil
}

func (c *Client) OperatorAddress() string { return c.kp.Address() }

// MemoHash — contract lưu hash của memo chứ không lưu memo. Memo chính là order id,
// công khai nó lên chain là công khai luôn mối nối giữa ví buyer và một đơn cụ thể.
func MemoHash(memo string) []byte {
	sum := sha256.Sum256([]byte(memo))
	return sum[:]
}

// CreateOrder — đăng ký order lên chain ở trạng thái mở.
func (c *Client) CreateOrder(ctx context.Context, order *models.OrderIntent) (string, error) {
	buyer, err := scAccount(order.BuyerWallet)
	if err != nil {
		return "", err
	}
	// Contract nhận i128 stroops; Stellar có đúng 7 chữ số thập phân.
	stroops := order.ExpectedAmount.Shift(libstellar.StellarDecimals).IntPart()
	return c.invoke(ctx, "create_order",
		scString(order.Id),
		buyer,
		scI128(stroops),
		scString(order.AssetCode),
		scBytes(MemoHash(order.Memo)),
	)
}

// SetStatus — ghi một bước chuyển trạng thái. Contract tự chặn bước nhảy sai, nên
// gọi sai thứ tự sẽ hỏng ở chain chứ không âm thầm ghi đè.
func (c *Client) SetStatus(ctx context.Context, orderId string, s models.OrderStatus, reason models.RejectReason) (string, error) {
	status, err := statusScVal(s, reason)
	if err != nil {
		return "", err
	}
	return c.invoke(ctx, "set_status", scString(orderId), status)
}

// invoke — simulate để lấy footprint/fee/auth, gắn lại vào transaction, ký, gửi, chờ kết quả.
// Soroban bắt buộc vòng simulate này: không có SorobanData từ simulation thì transaction
// không đủ resource và bị từ chối.
func (c *Client) invoke(ctx context.Context, fn string, args ...xdr.ScVal) (string, error) {
	acct, err := c.rpc.LoadAccount(ctx, c.kp.Address())
	if err != nil {
		return "", fmt.Errorf("load account operator: %w", err)
	}
	seq, err := acct.GetSequenceNumber()
	if err != nil {
		return "", err
	}

	build := func(fee int64, ext xdr.TransactionExt, auth []xdr.SorobanAuthorizationEntry) (*txnbuild.Transaction, error) {
		op := &txnbuild.InvokeHostFunction{
			HostFunction: xdr.HostFunction{
				Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
				InvokeContract: &xdr.InvokeContractArgs{
					ContractAddress: c.contract,
					FunctionName:    xdr.ScSymbol(fn),
					Args:            args,
				},
			},
			Auth:          auth,
			Ext:           ext,
			SourceAccount: c.kp.Address(),
		}
		// Dựng lại từ cùng một sequence: bản simulate và bản gửi thật phải là cùng
		// một transaction, chỉ khác fee/footprint.
		return txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: c.kp.Address(), Sequence: seq},
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{op},
			BaseFee:              fee,
			Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		})
	}

	tx, err := build(txnbuild.MinBaseFee, xdr.TransactionExt{V: 0}, nil)
	if err != nil {
		return "", err
	}
	raw, err := tx.Base64()
	if err != nil {
		return "", err
	}

	sim, err := c.rpc.SimulateTransaction(ctx, protocol.SimulateTransactionRequest{Transaction: raw})
	if err != nil {
		return "", fmt.Errorf("simulate %s: %w", fn, err)
	}
	if sim.Error != "" {
		return "", fmt.Errorf("simulate %s: %s", fn, sim.Error)
	}

	var sorobanData xdr.SorobanTransactionData
	if err := xdr.SafeUnmarshalBase64(sim.TransactionDataXDR, &sorobanData); err != nil {
		return "", fmt.Errorf("decode soroban data: %w", err)
	}
	var auth []xdr.SorobanAuthorizationEntry
	if len(sim.Results) > 0 && sim.Results[0].AuthXDR != nil {
		for _, entry := range *sim.Results[0].AuthXDR {
			var e xdr.SorobanAuthorizationEntry
			if err := xdr.SafeUnmarshalBase64(entry, &e); err != nil {
				return "", fmt.Errorf("decode auth entry: %w", err)
			}
			auth = append(auth, e)
		}
	}

	tx, err = build(txnbuild.MinBaseFee+sim.MinResourceFee, xdr.TransactionExt{V: 1, SorobanData: &sorobanData}, auth)
	if err != nil {
		return "", err
	}
	tx, err = tx.Sign(c.passphrase, c.kp)
	if err != nil {
		return "", err
	}
	signed, err := tx.Base64()
	if err != nil {
		return "", err
	}

	sent, err := c.rpc.SendTransaction(ctx, protocol.SendTransactionRequest{Transaction: signed})
	if err != nil {
		return "", fmt.Errorf("send %s: %w", fn, err)
	}
	if sent.Status == "ERROR" {
		return "", fmt.Errorf("send %s bị từ chối: %s", fn, sent.ErrorResultXDR)
	}

	// Chờ transaction vào ledger. Trả về hash kể cả khi lỗi, để log còn truy được.
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return sent.Hash, ctx.Err()
		case <-time.After(time.Second):
		}
		got, err := c.rpc.GetTransaction(ctx, protocol.GetTransactionRequest{Hash: sent.Hash})
		if err != nil {
			continue
		}
		switch got.Status {
		case protocol.TransactionStatusSuccess:
			return sent.Hash, nil
		case protocol.TransactionStatusFailed:
			return sent.Hash, fmt.Errorf("%s thất bại trên ledger: %s", fn, got.ResultXDR)
		}
	}
	return sent.Hash, fmt.Errorf("%s: hết thời gian chờ ledger", fn)
}

package libstellar

import (
	"fmt"
	"strings"

	"shopble/common/comtypes"
	"shopble/common/comutils"
	"shopble/config"

	"github.com/spf13/viper"
)

// Passphrase của từng network. Đây là thứ duy nhất phân biệt testnet với mainnet
// ở mức giao thức — ký sai passphrase thì transaction không được ledger chấp nhận.
const (
	TestnetPassphrase = "Test SDF Network ; September 2015"
	PublicPassphrase  = "Public Global Stellar Network ; September 2015"
)

// MemoTextMaxBytes — giới hạn cứng của MEMO_TEXT trong giao thức Stellar.
// Memo dài hơn 28 byte thì Horizon từ chối transaction, và buyer không có cách nào
// trả cho order đó nữa.
const MemoTextMaxBytes = 28

type StellarConfig struct {
	HorizonUrl        string `mapstructure:"horizon_url"`
	SorobanRpcUrl     string `mapstructure:"soroban_rpc_url"`
	NetworkPassphrase string `mapstructure:"network_passphrase"`

	// Stablecoin testnet cố định cho toàn bộ scope. Issuer phải ghi trong README
	// theo yêu cầu SOW Deliverable 1.
	AssetCode   string `mapstructure:"asset_code"`
	AssetIssuer string `mapstructure:"asset_issuer"`

	// Tài khoản nhận thanh toán. Watcher stream payment operations của đúng account này.
	DestinationAccount string `mapstructure:"destination_account"`

	// Soroban contract lưu trạng thái order. Rỗng cho tới khi deploy (Deliverable 3).
	ContractId string `mapstructure:"contract_id"`
	// Tên biến môi trường chứa secret key S... của operator ký set_status.
	// Secret KHÔNG nằm trong config file.
	OperatorSecretEnvVar string `mapstructure:"operator_secret_env_var"`
}

var vStellarConfig = comtypes.NewSingleton(func() *StellarConfig {
	var c StellarConfig
	comutils.PanicOnError(viper.UnmarshalKey(config.KeyStellarConfig, &c))
	if c.HorizonUrl == "" {
		c.HorizonUrl = "https://horizon-testnet.stellar.org"
	}
	if c.SorobanRpcUrl == "" {
		c.SorobanRpcUrl = "https://soroban-testnet.stellar.org"
	}
	if c.NetworkPassphrase == "" {
		c.NetworkPassphrase = TestnetPassphrase
	}
	comutils.PanicOnError(c.validate())
	return &c
})

func GetStellarConfig() *StellarConfig { return vStellarConfig.GetF() }

// validate — chặn cấu hình chạy được nhưng sai.
//
// Cổng mainnet là CỐ Ý và không có cờ tắt: SOW nói rõ toàn bộ scope là testnet và
// "no real value moves at any point". Một dòng config trỏ nhầm sang public network
// biến mọi payment instruction sinh ra thành lệnh chuyển tiền thật, và không có gì
// khác trong hệ thống phát hiện được điều đó. Rẻ hơn nhiều nếu app không khởi động nổi.
func (c *StellarConfig) validate() error {
	if c.NetworkPassphrase != TestnetPassphrase {
		return fmt.Errorf(
			"STELLAR_CONFIG.network_passphrase phải là testnet (%q), đang là %q — scope này testnet-only",
			TestnetPassphrase, c.NetworkPassphrase)
	}
	if strings.Contains(strings.ToLower(c.HorizonUrl), "horizon.stellar.org") {
		return fmt.Errorf("STELLAR_CONFIG.horizon_url trỏ vào mainnet Horizon: %q", c.HorizonUrl)
	}
	if c.AssetCode == "" || c.AssetIssuer == "" {
		return fmt.Errorf("STELLAR_CONFIG.asset_code và asset_issuer bắt buộc phải set")
	}
	if c.DestinationAccount == "" {
		return fmt.Errorf("STELLAR_CONFIG.destination_account bắt buộc phải set")
	}
	return nil
}

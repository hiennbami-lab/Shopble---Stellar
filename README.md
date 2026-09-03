# Shopble — Stablecoin Checkout & Order Reconciliation on Stellar (Phase 1)

Backend prototype cho Instawards Phase 1. Mục tiêu: một buyer đã có stablecoin trên
Stellar trả tiền cho **một order cụ thể**, và hệ thống chứng minh được — một cách xác
định, có thể kiểm chứng trên ledger — rằng payment đó thoả đúng order đó.

> **TESTNET ONLY.** Toàn bộ scope chạy trên Stellar testnet. Không có giá trị thật nào
> được chuyển. App **từ chối khởi động** nếu `STELLAR_CONFIG` trỏ vào public network
> (xem `project/shopble/lib/libstellar/config.go`).

## Asset

Scope dùng đúng **một** stablecoin testnet cố định. Điền vào `config.yml` và ghi lại ở đây:

| | |
|---|---|
| Asset code | `USDC` |
| Issuer | _(chưa set — điền `STELLAR_CONFIG.asset_issuer`)_ |
| Network passphrase | `Test SDF Network ; September 2015` |
| Horizon | `https://horizon-testnet.stellar.org` |
| Destination account | _(chưa set — điền `STELLAR_CONFIG.destination_account`)_ |

## Chạy local

```bash
# 1. Hạ tầng (postgres :5435, redis :6380)
docker compose up -d

# 2. Config
cp config.example.yml config.yml
#    điền asset_issuer + destination_account

# 3. Build. Tag `shopble` là BẮT BUỘC — thiếu nó binary không có lệnh nào.
go build --tags shopble -o shopble ./main.go

# 4. Migrate rồi chạy
CONFIG_PATH=./config.yml ./shopble migrate
CONFIG_PATH=./config.yml ./shopble api http --port 8080
```

- Health: <http://localhost:8080/health>
- Swagger: <http://localhost:8080/swagger/index.html>

Regen swagger sau khi đổi annotation:

```bash
GOFLAGS=-mod=mod go run github.com/swaggo/swag/cmd/swag@v1.16.6 init \
  -g project/shopble/services/api/swagger_anchor.go -o project/shopble/docs \
  --parseDependency --parseInternal --parseDepth 2
git checkout -- go.mod go.sum   # swag làm bẩn go.mod
```

## API hiện có

| Method | Path | Mô tả |
|---|---|---|
| `POST` | `/api/v1/orders` | Tạo order intent, trả về payment instruction (payload SEP-0007 + các field rời) |
| `GET` | `/api/v1/orders?buyer_wallet=G...` | Order của một ví |
| `GET` | `/api/v1/orders/{id}` | Một order kèm trạng thái |

Không có tài khoản user. **Ví đã connect chính là danh tính của buyer** — `buyer_wallet`
lưu trên order intent là thứ matcher đối chiếu với source account của payment.

## Data model

`shopble_order_intent` lưu đúng 7 field theo SOW Deliverable 1: product context,
expected amount, asset (code + issuer), destination account, memo, buyer wallet, status.

**Memo chính là order id.** ksuid dài 27 byte, vừa dưới trần 28 byte của `MEMO_TEXT`,
nên không cần bảng ánh xạ memo→order và không có chỗ nào để hai thứ lệch nhau.

State machine:

```
awaiting_payment → payment_detected → validated
                                    → rejected(reason)
```

Chỉ `validated` mới đủ điều kiện fulfilment. 5 lý do từ chối có tên:
`underpayment`, `wrong_asset`, `wrong_destination`, `wrong_buyer_wallet`, `invalid_memo`.

`shopble_payment_evidence` giữ bằng chứng thô đọc từ ledger (tx hash, source,
destination, asset, amount, memo, ledger close time) cộng verdict. `op_id` là unique —
đây là thứ duy nhất làm ingestion replay-safe khi watcher restart và stream lại từ
cursor cũ.

## Trạng thái so với SOW

| Deliverable | Trạng thái |
|---|---|
| 1 — Order intent + payment instruction | Backend xong. Còn wallet connect (Freighter / Stellar Wallets Kit) ở FE |
| 2 — Horizon watcher + matching engine | Data model + enum xong. Watcher và matcher chưa viết |
| 3 — Soroban order-status contract | Chưa bắt đầu. Config đã chừa `contract_id` + `operator_secret_env_var` |

## Layout

```
api/ common/ config/ database/ glib/   framework dùng chung
project/shopble/
  api/v1/         HTTP handlers + DTO
  cmd/            cobra commands (api, migrate)
  lib/libstellar/ config Stellar + payment instruction builder
  models/         GORM models + state machine
  services/api/   router + server bootstrap
```

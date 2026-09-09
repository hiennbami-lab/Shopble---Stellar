# Shopble — Stablecoin Checkout & Order Reconciliation on Stellar (Phase 1)

Backend prototype cho Instawards Phase 1. Mục tiêu: một buyer đã có stablecoin trên
Stellar trả tiền cho **một order cụ thể**, và hệ thống chứng minh được — một cách xác
định, có thể kiểm chứng trên ledger — rằng payment đó thoả đúng order đó.

> **TESTNET ONLY.** Toàn bộ scope chạy trên Stellar testnet. Không có giá trị thật nào
> được chuyển. App **từ chối khởi động** nếu `STELLAR_CONFIG` trỏ vào public network
> (xem `project/shopble/lib/libstellar/config.go`).

## Asset

Scope dùng đúng **một** stablecoin testnet cố định:

| | |
|---|---|
| Asset code | `USDC` |
| Issuer | `GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5` |
| Network passphrase | `Test SDF Network ; September 2015` |
| Horizon | `https://horizon-testnet.stellar.org` |
| Soroban RPC | `https://soroban-testnet.stellar.org` |
| Destination account | `GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ` |
| Order-status contract | `CCVC5G47V7EBMZNCRO2SBIJRD67RTMIZMOL75UEAIHCAF3HHXXNQNHCO` |
| Contract operator | `GDE7NZL663QM3CSG3V7XPIVSYFQXXI7VWSCSBGQAPLZ2MVSMOJFLQHDI` |

> **Destination account phải có trustline tới USDC/issuer ở trên.** Stellar không cho một
> account nhận non-native asset khi chưa có trustline: payment sẽ hỏng ngay trên ledger với
> `op_no_trust`, và không có gì trong backend sửa được chuyện đó.

## Chạy local

```bash
# 1. Hạ tầng (postgres :5435, redis :6380)
docker compose up -d

# 2. Config
cp config.example.yml config.yml
#    điền asset_issuer + destination_account

# 3. Build. Tag `shopble` là BẮT BUỘC — thiếu nó binary không có lệnh nào.
go build --tags shopble -o shopble ./main.go

# 4. Migrate
CONFIG_PATH=./config.yml ./shopble migrate

# 5. Secret của operator ký set_status (chỉ cần khi contract_id đã set).
#    KHÔNG nằm trong config file — tên biến do config.operator_secret_env_var chỉ ra.
export SHOPBLE_OPERATOR_SECRET=$(stellar keys secret shopble-operator)

# 6a. Chạy riêng từng phần
CONFIG_PATH=./config.yml ./shopble api http --port 8080
CONFIG_PATH=./config.yml ./shopble watch --interval 3

# 6b. Hoặc gộp API + watcher vào một process (một service trên VPS)
CONFIG_PATH=./config.yml ./shopble serve --port 8080 --interval 3

# 6c. Làm frontend, không cần ghi on-chain? Tắt hẳn, và phải nói ra:
CONFIG_PATH=./config.yml ./shopble serve --port 8080 --no-chain

# 7. Bảng kết quả nộp theo SOW, sinh từ dữ liệu đã lưu
CONFIG_PATH=./config.yml ./shopble report --out evidence-pack.md
```

Thiếu `SHOPBLE_OPERATOR_SECRET` trong khi `contract_id` đã set thì watcher **chết lúc boot**,
không im lặng bỏ qua. Bỏ qua âm thầm sẽ dẫn tới order `validated` trong Postgres nhưng không
bao giờ có trên chain — và chỉ lộ ra lúc đi tìm bằng chứng on-chain. Muốn chạy không chain thì
dùng `--no-chain`.

`DB_CONNECTION` đọc được từ biến môi trường (viper `AutomaticEnv`), nên khi deploy có thể
để config file không chứa credential nào và truyền connection string qua env.

- Health: <http://localhost:8080/health>
- Swagger: <http://localhost:8080/swagger/index.html>

Regen swagger sau khi đổi annotation:

> Đừng dùng `git checkout -- go.mod go.sum` để dọn: nó xoá luôn dependency mới thêm mà
> chưa commit. `go mod tidy` cho kết quả sạch tương đương mà không mất gì.

```bash
GOFLAGS=-mod=mod go run github.com/swaggo/swag/cmd/swag@v1.16.6 init \
  -g project/shopble/services/api/swagger_anchor.go -o project/shopble/docs \
  --parseDependency --parseInternal --parseDepth 2
go mod tidy                     # swag làm bẩn go.mod; tidy dọn lại
```

## API hiện có

| Method | Path | Mô tả |
|---|---|---|
| `POST` | `/api/v1/orders` | Tạo order intent, trả về payment instruction (payload SEP-0007 + các field rời) |
| `GET` | `/api/v1/orders?buyer_wallet=G...` | Order của một ví |
| `GET` | `/api/v1/orders/{id}` | Một order kèm trạng thái |
| `GET` | `/api/v1/orders/{id}/evidence` | Order + payment đã quan sát, so sánh **expected vs observed** từng field |
| `GET` | `/api/v1/evidence?verdict=&order_id=&limit=` | Evidence thô, kèm link Stellar Expert và detection latency |

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

## Watcher

Watcher **poll** `/accounts/{destination}/payments` của Horizon theo cursor, thay vì dùng SSE
như bản kế hoạch ban đầu mô tả. Quan sát được là như nhau, và cursor lưu trong DB làm cho việc
restart trở nên bình thường: `payment_evidence.op_id` là unique, nên đọc lại một đoạn ledger
chỉ tạo ra INSERT bị `ON CONFLICT DO NOTHING` nuốt, không đếm trùng payment. Cursor chỉ được
ghi trong cùng transaction với evidence.

Verdict ghi lên chain nằm NGOÀI transaction của Postgres, và hỏng thì chỉ log — Postgres mới
là bản ghi chuẩn. Nhưng một lần RPC hỏng mà không ai dọn sẽ thành lỗ vĩnh viễn trong bằng chứng
on-chain, nên mỗi vòng poll (sau khi đã bắt kịp tip) watcher quét lại các order đã chốt mà
`on_chain_tx_hash` còn rỗng và ghi lại, tối đa 3 lần mỗi order. Đây cũng là đường để order chốt
lúc chạy `--no-chain` sau này lên chain được. Order hết 3 lần vẫn hỏng thì hiện trong mục
Coverage của `shopble report`.

Lần chạy đầu tiên trên một account đã có lịch sử sẽ backfill toàn bộ payment cũ. Chúng đều
thành `rejected/invalid_memo` (không memo nào khớp order nào) — vô hại, nhưng `detection_latency_seconds`
của các dòng backfill là khoảng cách tới quá khứ, **không** phải độ trễ phát hiện thật. Chỉ
payment quan sát lúc watcher đang chạy mới có latency có nghĩa.

## Contract order-status (Soroban)

Nguồn: `contracts/order-status/`. Contract lưu **chỉ** trạng thái order — không giữ token,
không chuyển tiền, không escrow, không gọi contract khác.

```bash
cd contracts && cargo test          # unit test
stellar contract build              # ra target/wasm32v1-none/release/order_status.wasm
```

| Hàm | Ghi chú |
|---|---|
| `create_order(order_id, buyer, expected_amount, asset, memo_hash)` | Mở order ở `AwaitingPayment` |
| `set_status(order_id, status)` | Operator ký; contract tự chặn bước nhảy sai |
| `get_order(order_id)` | Đọc tự do, không cần auth |
| `is_fulfillable(order_id)` | `true` chỉ khi `Validated` |

Contract lưu **hash** của memo chứ không lưu memo: memo chính là order id, đưa nguyên lên
chain là công khai luôn mối nối giữa ví buyer và một đơn hàng cụ thể.

State machine được ép ở CẢ HAI phía. Contract từ chối `AwaitingPayment → Validated`
(lỗi `#3 InvalidTransition`), nên một bug ở backend không thể tự đánh dấu order đủ điều kiện
fulfilment khi chưa từng quan sát thấy payment nào.

## Evidence pack

```bash
CONFIG_PATH=./config.yml ./shopble report --out evidence-pack.md
```

Sinh thẳng ba bảng SOW bắt nộp từ dữ liệu đã lưu: 10 order intent run (7 field + payment
instruction sinh lại được), transaction của watcher (link Stellar Expert + detection latency),
và trạng thái on-chain của từng order đã chốt. Chép tay 25 dòng bảng thì sai một dòng cũng
không ai phát hiện, mà bảng chép tay thì không còn là bằng chứng.

Mục **Coverage** ở đầu báo cáo nói thẳng campaign còn thiếu gì: chưa đủ 10 run, chưa đủ 3 ví
khác nhau, lý do từ chối nào chưa có case, và order nào đã chốt trong Postgres mà chưa lên
chain. Cột `Expected` của bảng transaction để trống có chủ ý — ý định của test case không suy
ra được từ ledger, người chạy campaign tự điền.

## Trạng thái so với SOW

| Deliverable | Trạng thái |
|---|---|
| 1 — Order intent + payment instruction | Xong: backend, wallet connect, SEP-0007 + QR, và build-sign-submit qua ví. Còn: 10 run tài liệu hoá (cần trustline + ví có tiền) |
| 2 — Horizon watcher + matching engine | Xong: watcher, cursor, evidence, matcher 5 lý do, duplicate detection, API đọc evidence. Còn: 15 transaction của test campaign |
| 3 — Soroban order-status contract | Xong: contract + 10 unit test, deploy testnet, backend ghi verdict qua Soroban RPC, tự ghi lại khi RPC hỏng |
| 4 — Evidence pack | `shopble report` sinh sẵn bảng và mục coverage. Còn: chạy campaign, quay video, nộp |

## Layout

```
api/ common/ config/ database/ glib/   framework dùng chung
project/shopble/
  api/v1/         HTTP handlers + DTO
  cmd/            cobra commands (api, watch, serve, migrate, report)
  lib/libstellar/ config Stellar + payment instruction builder + Horizon client
  lib/libsoroban/ ghi verdict lên contract qua Soroban RPC
  services/watcher/ watcher + matching engine
  services/report/  sinh bảng evidence pack theo SOW
  web/            frontend (Vite + React)
contracts/order-status/  Soroban contract (Rust)
  models/         GORM models + state machine
  services/api/   router + server bootstrap
```

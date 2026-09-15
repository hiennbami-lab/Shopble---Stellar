# Shopble — Stablecoin Checkout & Order Reconciliation on Stellar (Phase 1)

Backend prototype for Instawards Phase 1. The goal: a buyer who already holds a stablecoin
on Stellar pays for **one specific order**, and the system can prove — deterministically,
and verifiably on the ledger — that the payment satisfies that order.

> **TESTNET ONLY.** The entire scope runs on Stellar testnet. No real value moves. The app
> **refuses to boot** if `STELLAR_CONFIG` points at the public network
> (see `project/shopble/lib/libstellar/config.go`).

## Asset

The scope uses exactly **one** fixed testnet stablecoin:

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

> **The destination account must hold a trustline to the USDC issuer above.** Stellar does not
> let an account receive a non-native asset without one: the payment fails on the ledger with
> `op_no_trust`, and nothing in the backend can fix that.

## Running locally

```bash
# 1. Infrastructure (postgres :5435, redis :6380 — redis is unused, see Deploy)
docker compose up -d

# 2. Config
cp config.example.yml config.yml
#    fill in asset_issuer + destination_account

# 3. Build. The `shopble` tag is MANDATORY — without it the binary has no commands at all.
go build --tags shopble -o shopble ./main.go

# 4. Migrate
CONFIG_PATH=./config.yml ./shopble migrate

# 5. Operator secret that signs set_status (only needed once contract_id is set).
#    NOT in the config file — the variable name comes from config.operator_secret_env_var.
export SHOPBLE_OPERATOR_SECRET=$(stellar keys secret shopble-operator)

# 6a. Run the pieces separately
CONFIG_PATH=./config.yml ./shopble api http --port 8080
CONFIG_PATH=./config.yml ./shopble watch --interval 3

# 6b. Or both in one process (one service unit on a VPS)
CONFIG_PATH=./config.yml ./shopble serve --port 8080 --interval 3

# 6c. Working on the frontend and don't need on-chain writes? Turn them off explicitly:
CONFIG_PATH=./config.yml ./shopble serve --port 8080 --no-chain

# 7. SOW submission tables, generated from stored data
CONFIG_PATH=./config.yml ./shopble report --out evidence-pack.md
```

If `SHOPBLE_OPERATOR_SECRET` is missing while `contract_id` is set, the watcher **dies at boot**
rather than skipping the write silently. Skipping silently would produce orders that are
`validated` in Postgres but never on chain — and that only surfaces when someone goes looking
for the on-chain evidence. To run without chain writes, say so with `--no-chain`.

`DB_CONNECTION` is read from the environment (viper `AutomaticEnv`), so a deployment can keep
credentials out of the config file entirely and pass the connection string as an env var. Note
that this only works for top-level keys: everything under `STELLAR_CONFIG` is read through
`viper.UnmarshalKey` and must live in the file.

- Health: <http://localhost:8080/health>
- Swagger: <http://localhost:8080/swagger/index.html>

Regenerate swagger after changing annotations:

> Don't clean up with `git checkout -- go.mod go.sum` — that also drops any new dependency you
> added but haven't committed. `go mod tidy` gives an equally clean result without losing it.

```bash
GOFLAGS=-mod=mod go run github.com/swaggo/swag/cmd/swag@v1.16.6 init \
  -g project/shopble/services/api/swagger_anchor.go -o project/shopble/docs \
  --parseDependency --parseInternal --parseDepth 2
go mod tidy                     # swag dirties go.mod; tidy puts it back
```

## Frontend

Vite + React, in `project/shopble/web/`. It talks to the backend over the same HTTP API
documented below, and builds the payment transaction in the browser for the connected wallet
to sign.

```bash
cd project/shopble/web
cp .env.example .env         # VITE_API_BASE, VITE_HORIZON
npm ci
npm run dev                  # :5173
npm run build                # -> dist/
npm test
```

Freighter only injects on **https** or `localhost`. A deployment served over plain http will
look fine and then fail to connect a wallet, so put TLS in front of it.

For a deployment the frontend ships as its own image (`docker/web/shopble-web.dockerfile`):
node builds the bundle, nginx serves it and proxies `/api` to the API container. See Deploy.

## Deploy

`shopble serve` runs the HTTP API and the watcher in **one process**, so a deployment is a
single service unit. `docker/app/shopble.dockerfile` builds a static binary onto distroless.

Two images, two services, both published on the host: `api` on `${API_PORT}` (8081) and `web`
on `${WEB_PORT}` (3000). A TLS-terminating nginx in front routes one hostname to both — `/api`
and `/health` to 8081, everything else to 3000 — which is what makes the app **same-origin** in
the browser and why the API's CORS middleware never comes into play.

The `web` container also proxies `/api` to `api` internally, so hitting it directly on port 3000
works on its own before that front nginx exists. Once the front nginx routes `/api` itself, the
internal proxy is redundant — harmless, and one `location` block in `docker/web/nginx.conf` to
delete if you want the web container to be purely static.

Vite inlines env vars at **build** time, so `VITE_API_BASE` and `VITE_HORIZON` are build args on
the web image — repointing the bundle means rebuilding it. Leave `VITE_API_BASE` **empty**: the
app then calls `/api` on whatever origin served it, which is correct both behind the front nginx
and when hitting port 3000 directly. Setting it to an absolute URL brings CORS back and pins the
bundle to one hostname.

Three things decide whether the backend actually works:

- **Run `serve`, not `api http`.** The image defaults to `serve`, which is the API and the
  watcher in one process. Overriding the command with `api http` starts the API with **no
  watcher** — nothing reads the ledger, no payment is ever detected, and `/health` still
  answers `healthy`. Only split the two when running more than one API replica.
- **Exactly one watcher instance.** The Horizon cursor is per-stream and the chain-write retry
  counter is an in-memory map. Two replicas double-process the ledger. Scale the API by running
  `api http` separately if that is ever needed; the watcher stays at one.
- **Keep `RELEASE_MODE` off `prod`**, or serve the frontend from the same origin. At
  `RELEASE_MODE: prod` the router drops the CORS middleware entirely and hides Swagger
  (`project/shopble/services/api/routes.go`), so a frontend on another origin stops working.

**Redis is not used.** `apiPrerun` and `watchPrerun` initialise the database and the Stellar
config, nothing else. The `CACHE_DB` / `LOCK_DB` blocks in the config file are parsed and never
dialled; a deployment can drop redis altogether and leave them in place.

### Docker Swarm stack

`docker-compose.yml` serves both paths. `docker stack deploy` ignores `build`, `container_name`
and `restart` — those are for local compose — and reads `deploy` for the swarm-side rules the
file already encodes: one replica, restart on any exit, `stop-first` updates.

```bash
cp .env.example .env         # SHOPBLE_IMAGE, SHOPBLE_CONFIG, DB_CONNECTION, SHOPBLE_OPERATOR_SECRET
set -a; . ./.env; set +a     # stack deploy interpolates from the shell, not from .env

docker build -f docker/app/shopble.dockerfile -t "$SHOPBLE_IMAGE" .
docker build -f docker/web/shopble-web.dockerfile -t "$SHOPBLE_WEB_IMAGE" \
  --build-arg VITE_API_BASE="$VITE_API_BASE" --build-arg VITE_HORIZON="$VITE_HORIZON" .
docker push "$SHOPBLE_IMAGE" && docker push "$SHOPBLE_WEB_IMAGE"
docker stack deploy -c docker-compose.yml shopble
```

`SHOPBLE_IMAGE` and `SHOPBLE_WEB_IMAGE` must be registry paths on swarm — stack deploy pulls,
it never builds.
`SHOPBLE_CONFIG` must be an **absolute** path on the node: swarm does not resolve relative bind
mounts. Locally, `docker compose up -d --build` works with the defaults as they ship.

Reusing an existing Postgres on a shared network: point `DB_CONNECTION` at the service name and
the internal port (`postgres://user:pass@<service>:5432/shopble?sslmode=disable`), not at any
published host port.

Migration has no one-shot equivalent in swarm; run it as a plain container on the same network
before the first deploy:

```bash
docker run --rm --network db_private_network \
  -e DB_CONNECTION="$DB_CONNECTION" \
  -v "$SHOPBLE_CONFIG":/etc/shopble/config.yml:ro \
  "$SHOPBLE_IMAGE" migrate
```

After deploying, `docker service logs -f shopble_api` should show
`watcher: stream="payments" dest=GA7Q… interval=3s`. If that line is absent, the command
override did not take and nothing is watching the ledger.

## API

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/orders` | Create an order intent, returns the payment instruction (SEP-0007 payload + the individual fields) |
| `GET` | `/api/v1/orders?buyer_wallet=G...` | Orders for one wallet |
| `GET` | `/api/v1/orders/{id}` | One order with its status |
| `GET` | `/api/v1/orders/{id}/evidence` | Order + observed payments, with a per-field **expected vs observed** comparison |
| `GET` | `/api/v1/evidence?verdict=&order_id=&limit=` | Raw evidence, including Stellar Expert links and detection latency |

There are no user accounts. **The connected wallet is the buyer's identity** — the
`buyer_wallet` stored on the order intent is what the matcher compares against the payment's
source account.

## Data model

`shopble_order_intent` stores exactly the 7 fields SOW Deliverable 1 asks for: product context,
expected amount, asset (code + issuer), destination account, memo, buyer wallet, status.

**The memo is the order id.** A ksuid is 27 bytes, just under the 28-byte `MEMO_TEXT` ceiling,
so there is no memo→order mapping table and nowhere for the two to drift apart.

State machine:

```
awaiting_payment → payment_detected → validated
                                    → rejected(reason)
```

Only `validated` qualifies for fulfilment. The five rejection reasons are named:
`underpayment`, `wrong_asset`, `wrong_destination`, `wrong_buyer_wallet`, `invalid_memo`.

`shopble_payment_evidence` keeps the raw evidence read off the ledger (tx hash, source,
destination, asset, amount, memo, ledger close time) plus the verdict. `op_id` is unique — that
is the single thing that makes ingestion replay-safe when the watcher restarts and re-reads from
an older cursor.

## Watcher

The watcher **polls** Horizon's `/accounts/{destination}/payments` by cursor rather than using
SSE as the original plan described. The observations are the same, and storing the cursor in the
database makes restarts unremarkable: `payment_evidence.op_id` is unique, so re-reading a stretch
of ledger produces INSERTs that `ON CONFLICT DO NOTHING` swallows instead of double-counting
payments. The cursor is only written inside the same transaction as the evidence.

Chain writes happen OUTSIDE the Postgres transaction, and a failure is only logged — Postgres is
the system of record. But one failed RPC call that nobody cleans up becomes a permanent hole in
the on-chain evidence, so on each poll cycle (once caught up to the tip) the watcher re-scans
settled orders whose `on_chain_tx_hash` is still empty and retries, at most 3 attempts per order.
This is also the path by which an order settled under `--no-chain` reaches the chain later. An
order that exhausts its attempts shows up in the Coverage section of `shopble report`.

Note that the retry counter lives in memory: restarting the watcher gives every order a fresh
set of attempts, which is the right trade — a failure that has since healed is worth retrying.

Both `payment` and `path_payment_strict_send` / `path_payment_strict_receive` are ingested. A
buyer routing through the DEX is still a buyer paying; for a path payment the `asset` / `amount`
Horizon returns are already what the destination **received**, so the matcher reads them exactly
as it reads a classic payment. Skipping them is a silent failure: the order sits in
`awaiting_payment` forever and nothing looks wrong.

The first run against an account with history backfills every old payment. They all come out as
`rejected/invalid_memo` (no memo matches any order) — harmless, but the
`detection_latency_seconds` on a backfilled row is the distance to the past, **not** real
detection latency. Only payments observed while the watcher is running have a meaningful number.

## Order-status contract (Soroban)

Source: `contracts/order-status/`. The contract stores **only** order state — it holds no
tokens, moves no money, is not an escrow, and calls no other contract.

```bash
cd contracts && cargo test          # unit tests
stellar contract build              # -> target/wasm32v1-none/release/order_status.wasm
```

| Function | Notes |
|---|---|
| `create_order(order_id, buyer, expected_amount, asset, memo_hash)` | Opens the order at `AwaitingPayment` |
| `set_status(order_id, status)` | Operator-signed; the contract rejects invalid transitions itself |
| `get_order(order_id)` | Unauthenticated read |
| `is_fulfillable(order_id)` | `true` only when `Validated` |

The contract stores a **hash** of the memo, not the memo: the memo is the order id, and putting
it on chain in the clear would publish the link between a buyer's wallet and a specific purchase.

The state machine is enforced on **both** sides. The contract rejects
`AwaitingPayment → Validated` (error `#3 InvalidTransition`), so a backend bug cannot mark an
order eligible for fulfilment when no payment was ever observed.

## Evidence pack

```bash
CONFIG_PATH=./config.yml ./shopble report --out evidence-pack.md
```

Generates the three tables the SOW requires straight from stored data: 10 order-intent runs
(7 fields + a regenerated payment instruction), the watcher's transactions (Stellar Expert link
+ detection latency), and the on-chain status of each settled order. Copying 25 table rows by
hand means a wrong row goes unnoticed — and a hand-copied table is not evidence any more.

The **Coverage** section at the top of the report states plainly what the campaign still lacks:
fewer than 10 runs, fewer than 3 distinct wallets, which rejection reasons have no case yet, and
which orders settled in Postgres without reaching the chain. The `Expected` column of the
transaction table is deliberately blank — a test case's intent cannot be derived from the ledger,
so whoever runs the campaign fills it in.

## Known limits

**On-chain entries expire after ~7 days.** Soroban charges for storage by time: every persistent
entry lives until a given ledger and is archived if nobody extends it. The contract does not call
`extend_ttl`, so each entry gets exactly the network minimum at write time — read off the
deployed contract: **120,960 ledgers ≈ 7 days** since the last write.

```bash
stellar contract read --id <contract_id> --durability persistent --network testnet
# the last column is liveUntilLedgerSeq
```

This is not hypothetical: the deployed contract instance was observed archived on 15 Sept 2026,
six days after deployment. **The backend cannot restore an archived entry** — `libsoroban` runs
simulate → sign → send and ignores the `restorePreamble` the simulation returns, so chain writes
fail until someone restores it from the CLI. Any invoke through `stellar contract invoke` restores
and extends automatically, so before a demo or an evidence run:

```bash
stellar contract invoke --id <contract_id> --source-account shopble-operator \
  --network testnet -- get_operator
```

Expiry does **not** damage evidence already submitted: Stellar Expert transaction links are ledger
history and last forever — only the re-readable state expires. Acceptable for a 30-day prototype;
a real deployment bumps TTL on every read and write. Changing the contract means redeploying under
a new contract id, which is why it is out of scope for Phase 1: the current id is already cited in
the evidence pack.

Postgres is the order's system of record — the chain is a verifiable copy, not the source.

## Status against the SOW

| Deliverable | Status |
|---|---|
| 1 — Order intent + payment instruction | Done: backend, wallet connect, SEP-0007 + QR, build-sign-submit through the wallet. Remaining: 10 documented runs (needs the trustline + funded wallets) |
| 2 — Horizon watcher + matching engine | Done: watcher, cursor, evidence, matcher with 5 reasons, duplicate detection, evidence read API, path payments. Remaining: the 15-transaction campaign |
| 3 — Soroban order-status contract | Done: contract + 10 unit tests, deployed to testnet, backend writes verdicts over Soroban RPC and retries failed writes |
| 4 — Evidence pack | `shopble report` generates the tables and the coverage section. Remaining: run the campaign, record the video, submit |

Everything still open needs operator action first: a USDC trustline on the destination account,
then 3+ funded testnet buyer wallets. Until the trustline exists, no order can reach `validated`.

## Layout

```
api/ common/ config/ database/ glib/   shared framework
project/shopble/
  api/v1/           HTTP handlers + DTOs
  cmd/              cobra commands (api, watch, serve, migrate, report)
  lib/libstellar/   Stellar config + payment instruction builder + Horizon client
  lib/libsoroban/   writes verdicts to the contract over Soroban RPC
  models/           GORM models + state machine
  services/api/     router + server bootstrap
  services/watcher/ watcher + matching engine
  services/report/  generates the SOW evidence-pack tables
  web/              frontend (Vite + React)
contracts/order-status/   Soroban contract (Rust)
docker/app/               distroless image for the backend service
docker/web/               node build + nginx image for the frontend
```

# syntax=docker/dockerfile:1

# ---------- builder ----------
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Deps before source: this layer only rebuilds when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 — static binary, runs on distroless static (which has no libc).
# The `shopble` tag is MANDATORY: without it the resulting binary has none of this
# project's commands (api/serve/watch/migrate/report all sit behind that build tag).
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -tags shopble -trimpath \
    -ldflags="-s -w" -o /out/shopble ./main.go

# ---------- runtime ----------
# static-debian12 ships ca-certificates (needed for Horizon and Soroban RPC) and a
# nonroot user. No shell, no package manager, nothing to exec into.
FROM gcr.io/distroless/static-debian12:nonroot

# CONFIG_PATH has no default in code — config.Init() reads this env var directly.
ENV CONFIG_PATH=/etc/shopble/config.yml

COPY --from=builder /out/shopble /usr/local/bin/shopble

WORKDIR /etc/shopble
USER nonroot:nonroot
EXPOSE 8080

# Do NOT COPY config.yml or the operator secret into the image — a pushed layer cannot
# be taken back:
#   config: -v ./config.yml:/etc/shopble/config.yml:ro
#   secret: -e SHOPBLE_OPERATOR_SECRET=...  (the orchestrator's secret manager)
#
# No HEALTHCHECK: the image has neither a shell nor curl, so the platform calls
# GET /health itself.
ENTRYPOINT ["/usr/local/bin/shopble"]

# `serve` = HTTP API + payment watcher in one process. This is the whole service, and
# it is the default on purpose: `api http` alone starts the API with nothing reading the
# ledger, which looks healthy and silently detects no payments at all. Override the
# command to split the two only when running more than one API replica.
CMD ["serve", "--host", "0.0.0.0", "--port", "8080", "--interval", "3"]

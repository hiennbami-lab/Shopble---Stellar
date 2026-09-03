# syntax=docker/dockerfile:1

# ---------- builder ----------
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Deps trước source: layer này chỉ rebuild khi go.mod/go.sum đổi.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 — binary tĩnh, chạy được trên distroless static (không có libc).
# Tag `shopble` là BẮT BUỘC: thiếu nó thì binary build ra không có lệnh nào của
# project này (api/migrate đều nằm sau build tag đó).
RUN CGO_ENABLED=0 GOOS=linux go build -tags shopble -trimpath \
    -ldflags="-s -w" -o /out/shopble ./main.go

# ---------- runtime ----------
# static-debian12 có sẵn ca-certificates (cần cho Horizon và Soroban RPC) và user nonroot.
FROM gcr.io/distroless/static-debian12:nonroot

# CONFIG_PATH không có default trong code — config.Init() đọc thẳng env này.
ENV CONFIG_PATH=/etc/shopble/config.yml

COPY --from=builder /out/shopble /usr/local/bin/shopble

WORKDIR /etc/shopble
USER nonroot:nonroot
EXPOSE 8080

# KHÔNG COPY config.yml hay operator secret vào image — một layer đã push thì không
# rút lại được:
#   config: -v ./config.yml:/etc/shopble/config.yml:ro
#   secret: -e SHOPBLE_OPERATOR_SECRET=...  (secret manager của orchestrator)
#
# Healthcheck: image không có shell lẫn curl, nên để platform gọi GET /health.
ENTRYPOINT ["/usr/local/bin/shopble"]
CMD ["api", "http", "--host", "0.0.0.0", "--port", "8080"]

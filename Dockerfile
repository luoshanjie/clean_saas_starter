FROM golang:1.25-alpine AS builder

WORKDIR /src

# Enable module cache layer first.
COPY go.mod go.sum ./
RUN go mod download

# Build service binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/service

# Runtime image (root user for host-mounted log dir compatibility).
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache tzdata ca-certificates \
    && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo Asia/Shanghai > /etc/timezone

COPY --from=builder /out/service /app/service
# Casbin model/policy are loaded from relative file paths at runtime.
COPY --from=builder /src/internal/repo/casbin/model.conf /app/internal/repo/casbin/model.conf
COPY --from=builder /src/internal/repo/casbin/policy.csv /app/internal/repo/casbin/policy.csv

ENV ADDR=:9527
EXPOSE 9527

ENTRYPOINT ["/app/service"]

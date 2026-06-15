FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o qianwen2api main.go

FROM chromedp/headless-shell:latest

RUN apt-get update && \
    apt-get install -y ca-certificates dumb-init && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -r appuser && useradd -r -g appuser appuser

WORKDIR /app

COPY --from=builder /app/qianwen2api .

RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

ENV HOST=0.0.0.0 \
    PORT=8000 \
    DATA_DIR=/app/data \
    DATABASE_PATH=/app/data/qianwen-web-01.sqlite

EXPOSE 8000

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["./qianwen2api"]

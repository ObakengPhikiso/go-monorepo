FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY libs/shared/ ../libs/shared/
COPY services/auth/go.mod ./
COPY services/auth/ .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o auth .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/auth .

EXPOSE 8084

HEALTHCHECK --interval=10s --timeout=2s --start-period=5s --retries=3 \
  CMD wget --spider -q http://localhost:8084/health || exit 1

CMD ["./auth"]

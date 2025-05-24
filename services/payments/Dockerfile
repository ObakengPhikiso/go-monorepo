# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
# Copy go.mod and go.sum for dependency resolution
COPY services/payments/go.mod ./
COPY services/payments/go.sum ./
# Copy shared library
COPY libs/shared/ ../libs/shared/
# Copy service source
COPY services/payments/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o payments .

# Final stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/payments .
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=2s --start-period=5s --retries=3 \
  CMD wget --spider -q http://localhost:8080/health || exit 1
CMD ["./payments"]

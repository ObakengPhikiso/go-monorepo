# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
# Copy go.mod and go.sum for dependency resolution
COPY services/orders/go.mod ./
COPY services/orders/go.sum ./
# Copy shared library
COPY libs/shared/ ../libs/shared/
# Copy service source
COPY services/orders/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o orders .

# Final stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/orders .
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=2s --start-period=5s --retries=3 \
  CMD wget --spider -q http://localhost:8080/health || exit 1
CMD ["./orders"]

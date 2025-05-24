# Build stage
FROM golang:1.20-alpine AS builder
WORKDIR /app
# Copy go.mod and go.sum for dependency resolution
COPY services/api-gateway/go.mod ./
# Copy shared library
COPY libs/shared/ ../libs/shared/
# Copy service source
COPY services/api-gateway/ .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway .

# Final stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/api-gateway .
COPY services/api-gateway/docs/swagger.yaml ./docs/swagger.yaml
EXPOSE 8088
CMD ["./api-gateway"]

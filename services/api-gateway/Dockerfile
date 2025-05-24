# Build stage
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod init github.com/ObakengPhikiso/monorepo/services/api-gateway || true
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway .

# Final stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/api-gateway .
EXPOSE 8088
CMD ["./api-gateway"]

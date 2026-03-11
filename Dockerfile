FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o futbol-app ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/futbol-app .
COPY --from=builder /app/internal/interface/templates ./internal/interface/templates
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./futbol-app"]

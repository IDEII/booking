FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/server ./cmd/server/main.go

FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache tzdata ca-certificates

COPY --from=builder /app/server .

COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
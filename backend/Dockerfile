FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/api

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /server /server
COPY db/migrations /migrations

EXPOSE 8080

CMD ["/server"]

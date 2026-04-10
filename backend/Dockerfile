FROM golang:1.24-alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/api

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server /app/server
COPY --from=builder /app/db/migrations /app/db/migrations
COPY --from=builder /app/internal/i18n/locales /app/internal/i18n/locales

EXPOSE 8080

CMD ["/app/server"]

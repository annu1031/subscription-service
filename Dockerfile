FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subscription-api ./cmd/api

FROM alpine:3.15

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=UTC

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/subscription-api .

USER appuser

EXPOSE 8080

CMD ["./subscription-api"]
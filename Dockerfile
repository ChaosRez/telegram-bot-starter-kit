FROM golang:1.26.6-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/telegram-bot \
    ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/telegram-bot /app/telegram-bot
COPY config/config.yaml /app/config/config.yaml

ENTRYPOINT ["/app/telegram-bot"]

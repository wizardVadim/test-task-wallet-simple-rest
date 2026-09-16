FROM golang:1.26-bookworm

WORKDIR /app

COPY services/gw-currency-wallet/go.mod services/gw-currency-wallet/go.sum ./
RUN go mod download

COPY services/gw-currency-wallet/ ./
RUN go build -o /app/exe ./cmd

CMD ["/app/exe"]
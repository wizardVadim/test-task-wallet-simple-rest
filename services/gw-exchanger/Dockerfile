FROM golang:1.27.1-bookworm

ENV GOWORK=off
WORKDIR /app/services/gw-exchanger

COPY contracts/ /app/contracts/
COPY services/gw-exchanger/go.mod services/gw-exchanger/go.sum ./
RUN go mod download

COPY services/gw-exchanger/ ./
RUN go build -o /app/exe ./cmd

CMD ["/app/exe"]

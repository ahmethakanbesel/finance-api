FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /finance-api ./cmd/finance-api

FROM gcr.io/distroless/static-debian12
COPY --from=builder /finance-api .
EXPOSE 8080
CMD ["/finance-api"]

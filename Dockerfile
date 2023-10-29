FROM golang:1.21-alpine as builder
WORKDIR /app

COPY . ./
RUN CGO_ENABLED=0 go build -o /finance-api

FROM gcr.io/distroless/static-debian12
COPY --from=builder /finance-api .
EXPOSE 8080
CMD ["/finance-api", "serve", "--http=0.0.0.0:8080"]
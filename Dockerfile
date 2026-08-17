FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/configguard ./cmd/configguard

FROM alpine:3.20
RUN adduser -D -u 10001 configguard
COPY --from=builder /out/configguard /usr/local/bin/configguard
USER configguard
EXPOSE 8080 9090
ENTRYPOINT ["configguard"]
CMD ["serve", "--http", ":8080", "--grpc", ":9090"]

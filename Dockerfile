FROM golang:1.23 AS builder
WORKDIR /app
COPY src/* ./
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main /app/main
COPY mailrelay.json /etc/mailrelay/mailrelay.json
CMD ["/app/main", "-config", "/etc/mailrelay/mailrelay.json"]

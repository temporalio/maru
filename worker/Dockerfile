# Stage 1: Build
FROM golang:1.16-alpine3.12 AS builder

# Install build tools
RUN apk add --update --no-cache ca-certificates git make openssh

# Making sure that dependency is not touched
ENV GOFLAGS="-mod=readonly"

WORKDIR /temporal-bench

# Copy go mod dependencies and build cache
COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

# need to make clean first in case binaries to be built are stale
RUN make clean && CGO_ENABLED=0 make bins

# Stage 2: Bench worker
FROM alpine:3.12 AS bench-worker

COPY --from=builder /temporal-bench/bins/temporal-bench /usr/local/bin/

ENV NAMESPACE default
ENV FRONTEND_ADDRESS 127.0.0.1:7233
ENV NUM_DECISION_POLLERS 10
ENV TLS_CA_CERT_FILE ""
ENV TLS_CLIENT_CERT_FILE ""
ENV TLS_CLIENT_CERT_PRIVATE_KEY_FILE ""

# Base64 equivalents of above
ENV TLS_CA_CERT_DATA ""
ENV TLS_CLIENT_CERT_DATA ""
ENV TLS_CLIENT_CERT_PRIVATE_KEY_DATA ""
ENV TLS_ENABLE_HOST_VERIFICATION "false"

ENTRYPOINT ["/usr/local/bin/temporal-bench"]

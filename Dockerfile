################################################################################
# BUILDER/DEVELOPMENT IMAGE
################################################################################

FROM golang:1.17.0-alpine as builder

# Install Git
RUN apk add --no-cache git libc6-compat make

# go build will fail in alpine if this is enabled as it looks for gcc
ENV CGO_ENABLED 0

WORKDIR /build/

COPY go.mod go.sum /build/

RUN go mod download

COPY . /build/

# Build the executable
RUN go build -o deluge-telegraf-plugin cmd/main.go

################################################################################
# LINT IMAGE
################################################################################

FROM golang:1.17.0 as ci

# Install golangci
RUN curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.39.0

WORKDIR /app

COPY --from=builder /build .

COPY .golangci.yaml .

################################################################################
# FINAL IMAGE
################################################################################

FROM telegraf:1.19-alpine

COPY --from=builder /build/deluge-telegraf-plugin /app/

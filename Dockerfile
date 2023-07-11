################################################################################
# BUILDER/DEVELOPMENT IMAGE
################################################################################

FROM golang:1.20.6-alpine as builder

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

FROM golang:1.20.6 as ci

# Install golangci
RUN curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.52.2

WORKDIR /app

COPY --from=builder /build .

COPY .golangci.yaml .

ENTRYPOINT [ "sh", "-c" ]
CMD [ "golangci-lint run -v && go test ./... -race -timeout 30m -p 1" ]

################################################################################
# FINAL IMAGE
################################################################################

FROM telegraf:1.27-alpine

COPY --from=builder /build/deluge-telegraf-plugin /app/

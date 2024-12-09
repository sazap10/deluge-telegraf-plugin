################################################################################
# BUILDER/DEVELOPMENT IMAGE
################################################################################

FROM golang:1.23-alpine AS builder

# Install Git
RUN apk add --no-cache git libc6-compat make

# go build will fail in alpine if this is enabled as it looks for gcc
ENV CGO_ENABLED=0

WORKDIR /build/

COPY go.mod go.sum /build/

RUN go mod download

COPY cmd /build/cmd
COPY plugins /build/plugins

# Build the executable
RUN go build -o deluge-telegraf-plugin cmd/main.go

################################################################################
# LINT IMAGE
################################################################################

FROM golang:1.23 AS ci

# Install golangci
RUN curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.62.2

WORKDIR /app

COPY --from=builder /build .

COPY .golangci.yml .

ENTRYPOINT [ "sh", "-c" ]
CMD [ "golangci-lint run -v && go test ./... -race -timeout 30m -p 1" ]

################################################################################
# FINAL IMAGE
################################################################################

FROM telegraf:1.32-alpine

RUN apk add --no-cache smartmontools nvme-cli ipmitool sudo && \
    echo 'telegraf ALL=NOPASSWD:/usr/sbin/smartctl *' | tee /etc/sudoers.d/telegraf && \
    echo 'telegraf ALL=NOPASSWD:/usr/sbin/nvme *'     | tee -a /etc/sudoers.d/telegraf && \
    echo 'telegraf ALL=NOPASSWD:/usr/sbin/ipmitool *'  | tee -a /etc/sudoers.d/telegraf

COPY --from=builder /build/deluge-telegraf-plugin /app/

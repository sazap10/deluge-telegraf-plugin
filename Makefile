GOOS ?= linux
GOARCH ?= amd64
BUILD_ARGS ?= 
build:
			GOOS=$(GOOS) GOARCH=$(GOARCH) $(BUILD_ARGS) go build -o bin/deluge-telegraf-plugin cmd/main.go
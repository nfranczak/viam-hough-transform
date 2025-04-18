GO_BUILD_ENV :=
GO_BUILD_FLAGS :=
MODULE_BINARY := bin/viam-hough-transform

ifeq ($(VIAM_TARGET_OS), windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	GO_BUILD_FLAGS := -tags no_cgo	
	MODULE_BINARY = bin/viam-hough-transform.exe
endif

ifeq ($(VIAM_TARGET_OS), linux)
	GO_BUILD_ENV += CGO_LDFLAGS='-ltbb'
	GO_BUILD_FLAGS := -tags opencvstatic
endif

$(MODULE_BINARY): Makefile go.mod cmd/module/*.go hough/*.go
	$(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $(MODULE_BINARY) cmd/module/main.go

lint:
	gofmt -s -w .

update:
	go get go.viam.com/rdk@latest
	go mod tidy

test:
	$(GO_BUILD_ENV) go test ./...

module.tar.gz: meta.json $(MODULE_BINARY)
ifeq ($(VIAM_TARGET_OS), windows)
	jq '.entrypoint = "./bin/viam-hough-transform.exe"' meta.json > temp.json && mv temp.json meta.json
else
	strip $(MODULE_BINARY)
endif
	tar czf $@ meta.json $(MODULE_BINARY)
ifeq ($(VIAM_TARGET_OS), windows)
	git checkout meta.json
endif

module: test module.tar.gz

all: test module.tar.gz

setup:


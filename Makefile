VERSION?="0.1.0"
DEST?=./bin

default: install

test:
	echo "==> Running tests..."
	go test -cover -v `go list ./... | grep -v /vendor/`

deps:
	echo "==> Install dependencies..."
	go get -u github.com/jteeuwen/go-bindata/...

build-configs:
	echo "==> Build configs..."
	${GOPATH}/bin/go-bindata -pkg config -o config/reference.go config/*.yml

build: build-configs
	echo "==> Build binaries..."
	go build -v -ldflags "-s -w -X main.version=${VERSION}" -o ${DEST}/serve-server main.go

install: test build
	echo "==> Copy binaries to \$GOPATH/bin/..."
	cp ${DEST}/* ${GOPATH}/bin/

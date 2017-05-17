VERSION?=$(shell git describe --tags --abbrev=0 | sed 's/v//')
DEST?=./bin
TAG="servehub/serve-server"

export CGO_ENABLED=0

default: install

test:
	@echo "==> Running tests..."
	go test -cover -v `go list ./... | grep -v /vendor/`

deps:
	@echo "==> Install dependencies..."
	go get github.com/jteeuwen/go-bindata/...
	go get github.com/Masterminds/glide
	glide i -v

clean:
	@echo "==> Cleanup old binaries..."
	rm -f ${DEST}/*

build-configs:
	@echo "==> Build configs..."
	${GOPATH}/bin/go-bindata -pkg config -o config/reference.go config/*.yml

build: build-configs
	@echo "==> Build binaries..."
	go build -v -ldflags "-s -w -X main.version=${VERSION}" -o ${DEST}/serve-server main.go

install: test build
	@echo "==> Copy binaries to \$GOPATH/bin/..."
	cp ${DEST}/* ${GOPATH}/bin/

dist: clean
	GOOS=linux GOARCH=amd64 make build

bump-tag:
	TAG=$$(echo "v${VERSION}" | awk -F. '{$$NF = $$NF + 1;} 1' | sed 's/ /./g'); \
	git tag $$TAG; \
	git push --tags

release: dist
	@echo "==> Build and publish new docker image..."
	docker build -t ${TAG}:latest -t ${TAG}:${VERSION} .
	docker push ${TAG}:${VERSION}
	docker push ${TAG}:latest

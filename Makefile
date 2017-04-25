VERSION?="0.2.5"
DEST?=./bin

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
	GOOS=linux make build

release: dist
	@echo "==> Build and publish new docker image..."
	docker build -t servehub/serve-server:latest -t servehub/serve-server:${VERSION} .
	docker push servehub/serve-server:${VERSION}
	docker push servehub/serve-server:latest

travis-release:
	@echo "==> Build and publish new docker image..."
	@docker login -u="${DOCKER_USERNAME}" -p="${DOCKER_PASSWORD}"
	docker build -t servehub/serve-server:latest -t servehub/serve-server:${VERSION} .
	docker push servehub/serve-server:${VERSION}
	docker push servehub/serve-server:latest
	docker logout

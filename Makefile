.PHONY: check test build image

check:
	bash tests/repository-config.test.sh
	bash tests/repository-hygiene.test.sh
	bash tests/container-layout.test.sh

test:
	go test -tags unittest ./...

build:
	go build -trimpath -o bin/blockbook blockbook.go

image:
	docker build --tag organic-blockbook:local .

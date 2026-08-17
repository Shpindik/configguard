BINARY := configguard
PROTO := api/proto/configguard/v1/configguard.proto

.PHONY: build run test vet fmt proto docker-build clean

build:
	go build -o bin/$(BINARY) ./cmd/configguard

run: build
	./bin/$(BINARY) $(ARGS)

test:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w .

proto:
	protoc --go_out=. --go_opt=module=configguard \
		--go-grpc_out=. --go-grpc_opt=module=configguard \
		$(PROTO)

docker-build:
	docker build -t $(BINARY) .

clean:
	rm -rf bin

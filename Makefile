.PHONY: build run test clean vet

BINARY := csm
CMD := ./cmd/csm

build: vet
	go build -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

test:
	go test ./... -cover -count=1

clean:
	rm -f $(BINARY)
	go clean

vet:
	go vet ./...

build:
	go build -o ./bin/kkrh-meta ./cmd/server/
	go build -o ./bin/processor ./cmd/processor

test:
	go test -v ./...
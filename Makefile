build:
	@go build -o bin/bank-app-go

run: build
	@./bin/bank-app-go

test:
	@go test -v ./...
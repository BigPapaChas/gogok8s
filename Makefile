lint:
	golangci-lint run

test:
	go test ./...

update-deps:
	go get -u ./...

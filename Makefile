lint:
	golangci-lint run

update-deps:
	go get -u ./...

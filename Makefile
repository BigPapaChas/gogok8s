lint:
	gofumpt -l -w .
	golangci-lint run --enable-all --disable wrapcheck,goerr113,gochecknoglobals,exhaustivestruct,gochecknoinits,interfacer,prealloc --fix
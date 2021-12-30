lint:
	gofumpt -l -w .
	goimports -w -local github.com/BigPapaChas/gogok8s .
	golangci-lint run --enable-all --disable wrapcheck,goerr113,gochecknoglobals,exhaustivestruct,gochecknoinits,interfacer,prealloc,gci,cyclop

FILES = $(shell find . -type f -name '*.go')


gofmt:
	@actools gofmt -w $(FILES)
	@actools gofmt -r '&α{} -> new(α)' -w $(FILES)

test:
	@actools go test ./... -src /workspace/testdata/src -output /workspace/tmp/output -data /workspace/testdata/data -watch false

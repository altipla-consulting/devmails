
FILES = $(shell find . -type f -name '*.go')


gofmt:
	@actools gofmt -w $(FILES)
	@actools gofmt -r '&α{} -> new(α)' -w $(FILES)

test:
	@actools go test ./... -src testdata/src -data testdata/data -watch=false

run:
	actools go install ./cmd/devmails
	actools run go devmails -src cmd/devmails/testdata/src -data cmd/devmails/testdata/data

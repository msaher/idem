.PHONY: test
test: gen
	go test -v -count=1 .

.PHONY: gen
gen:
	go run ./compile
	@# go generate

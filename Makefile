
# https://github.com/yandex/pandora/blob/dev/Makefile
# https://github.com/yandex/mysync/blob/master/Makefile

.PHONY: all build test lint vet fmt travis coverage checkfmt prepare deps

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

all: build test vet checkfmt

travis: build test checkfmt coverage

prepare: fmt build test vet

build:
	@echo "$(OK_COLOR)Build app$(NO_COLOR)"
	gofmt -s -w . 2>&1
	go mod download
	go build -o main.exe ./cmd/main.go

test:
	@echo "$(OK_COLOR)Test packages$(NO_COLOR)"
	go test -race -v ./...

coverage:
#	@echo "$(OK_COLOR)Make coverage report$(NO_COLOR)"
#	@./scripts/coverage.sh
#	-goveralls -coverprofile=gover.coverprofile -service=travis-ci

vet:
	@echo "$(OK_COLOR)Run vet$(NO_COLOR)"
	@go vet ./...

checkfmt:
#	@echo "$(OK_COLOR)Check formats$(NO_COLOR)"
#	@./scripts/checkfmt.sh .

fmt:
	@echo "$(OK_COLOR)Check fmt$(NO_COLOR)"
	@echo "FIXME go fmt does not format imports, should be fixed"
	@go fmt

tools:
	@echo "$(OK_COLOR)Install tools$(NO_COLOR)"
	go install golang.org/x/tools/cmd/goimports@latest
	go get golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls

deps:
	$(info #Install dependencies...)
	go mod tidy
	go mod download
